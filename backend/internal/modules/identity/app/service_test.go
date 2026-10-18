package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func signUpInput() app.SignUpInput {
	return app.SignUpInput{
		TenantName: "Ferretería El Tornillo",
		NIT:        validNIT,
		OwnerName:  "Ana Gómez",
		Email:      "Ana@ElTornillo.co",
		Password:   validPassword,

		BusinessTemplate: "ferreteria",
	}
}

// requireUsableRefresh checks that raw is the current refresh token of userID.
func requireUsableRefresh(t *testing.T, e *env, raw string, userID uuid.UUID) {
	t.Helper()
	tenantID, hash, err := domain.ParseRefreshToken(raw)
	require.NoError(t, err)
	stored, err := e.tokens.GetByHash(t.Context(), tenantID, hash)
	require.NoError(t, err)
	require.Equal(t, userID, stored.UserID())
	require.False(t, stored.IsRevoked())
}

func TestSignUp(t *testing.T) {
	t.Run("crea el tenant y el usuario owner y devuelve la sesión", func(t *testing.T) {
		e := newEnv(t)

		session, err := e.svc.SignUp(t.Context(), signUpInput())
		require.NoError(t, err)

		user, tenant := session.Profile.User, session.Profile.Tenant
		require.Equal(t, domain.RoleOwner, user.Role())
		require.Equal(t, "ana@eltornillo.co", user.Email().String())
		require.Equal(t, "Ana Gómez", user.Name())
		require.Equal(t, tenant.ID(), user.TenantID())
		require.Equal(t, "Ferretería El Tornillo", tenant.Name())
		require.Equal(t, validNIT, tenant.NIT().String())
		require.True(t, tenant.IsActive())

		_, err = e.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
		stored, err := e.users.GetByID(t.Context(), tenant.ID(), user.ID())
		require.NoError(t, err)
		require.Equal(t, "hash:"+validPassword, stored.PasswordHash(), "only the hash is stored")

		require.Equal(t, "access:"+tenant.ID().String()+":"+user.ID().String()+":owner", session.AccessToken)
		require.Equal(t, e.clock.Now().Add(15*time.Minute), session.AccessExpiresAt)
		require.Equal(t, e.clock.Now().Add(domain.RefreshTokenTTL), session.RefreshExpiresAt)
		requireUsableRefresh(t, e, session.RefreshToken, user.ID())

		require.EqualValues(t, 1, e.counter(t, "cuadra.identity.signups", "", ""))
		span := e.span(t, "identity.SignUp")
		require.Contains(t, span.Attributes(), attribute.String("tenant.id", tenant.ID().String()))
		require.Contains(t, span.Attributes(), attribute.String("user.id", user.ID().String()))
	})

	t.Run("el registro aplica la plantilla elegida dentro de su transacción", func(t *testing.T) {
		for _, template := range []string{"ferreteria", "ropa_calzado"} {
			t.Run(template, func(t *testing.T) {
				e := newEnv(t)
				in := signUpInput()
				in.BusinessTemplate = template

				session, err := e.svc.SignUp(t.Context(), in)
				require.NoError(t, err)

				require.Equal(t, template, session.Profile.Tenant.BusinessTemplate())
				stored, err := e.tenants.GetByID(t.Context(), session.Profile.Tenant.ID())
				require.NoError(t, err)
				require.Equal(t, template, stored.BusinessTemplate())
				require.Equal(t, []uuid.UUID{session.Profile.Tenant.ID()}, e.setup.tenants)
				require.Equal(t, []string{template}, e.setup.templates)
				require.Equal(t, []bool{true}, e.setup.inTx)
				require.EqualValues(t, 1, e.counter(t, "cuadra.identity.signups", "business_template", template))
				require.Contains(t, e.span(t, "identity.SignUp").Attributes(), attribute.String("business_template", template))
			})
		}
	})

	t.Run("el registro guarda el IVA por defecto de la plantilla", func(t *testing.T) {
		tests := []struct{ template, rate string }{
			{template: "ferreteria", rate: "0.19"},
			{template: "drogueria", rate: "0"},
		}
		for _, tt := range tests {
			t.Run(tt.template, func(t *testing.T) {
				e := newEnv(t)
				in := signUpInput()
				in.BusinessTemplate = tt.template

				session, err := e.svc.SignUp(t.Context(), in)
				require.NoError(t, err)

				require.Equal(t, tt.rate, session.Profile.Tenant.DefaultTaxRate().String())
				stored, err := e.tenants.GetByID(t.Context(), session.Profile.Tenant.ID())
				require.NoError(t, err)
				require.Equal(t, tt.rate, stored.DefaultTaxRate().String())
			})
		}
	})

	t.Run("un registro sin plantilla o con una inexistente falla sin crear el tenant", func(t *testing.T) {
		for _, template := range []string{"", "panaderia", "Ferreteria", "../generica"} {
			t.Run(template, func(t *testing.T) {
				e := newEnv(t)
				in := signUpInput()
				in.BusinessTemplate = template
				hashes := e.hasher.Hashes()

				_, err := e.svc.SignUp(t.Context(), in)

				require.ErrorIs(t, err, domain.ErrUnknownBusinessTemplate)
				require.Empty(t, e.tenants.byID)
				require.Empty(t, e.users.byID)
				require.Empty(t, e.setup.tenants)
				require.Equal(t, hashes, e.hasher.Hashes(), "the password is not hashed")
			})
		}
	})

	t.Run("lista las plantillas disponibles", func(t *testing.T) {
		e := newEnv(t)
		require.Equal(t, testTemplates, e.svc.BusinessTemplates())
	})

	t.Run("si la preparación del tenant falla, el registro falla y no crea el tenant", func(t *testing.T) {
		e := newEnv(t)
		e.setup.err = errors.New("catalog down")

		_, err := e.svc.SignUp(t.Context(), signUpInput())

		require.ErrorIs(t, err, e.setup.err)
		require.Empty(t, e.tenants.byID)
		require.Empty(t, e.users.byID)
		require.Empty(t, e.tokens.byID)
		require.Zero(t, e.counter(t, "cuadra.identity.signups", "", ""))
	})

	t.Run("rechaza datos inválidos sin crear el tenant", func(t *testing.T) {
		tests := []struct {
			name    string
			mutate  func(*app.SignUpInput)
			wantErr error
		}{
			{name: "email inválido", mutate: func(in *app.SignUpInput) { in.Email = "ana@" }, wantErr: domain.ErrInvalidEmail},
			{name: "NIT con DV incorrecto", mutate: func(in *app.SignUpInput) { in.NIT = "890903938-1" }, wantErr: domain.ErrInvalidNITCheckDigit},
			{name: "contraseña corta", mutate: func(in *app.SignUpInput) { in.Password = "corta" }, wantErr: domain.ErrPasswordTooShort},
			{name: "contraseña larga", mutate: func(in *app.SignUpInput) { in.Password = string(make([]byte, 73)) }, wantErr: domain.ErrPasswordTooLong},
			{name: "empresa sin nombre", mutate: func(in *app.SignUpInput) { in.TenantName = "  " }, wantErr: domain.ErrInvalidTenantName},
			{name: "dueño sin nombre", mutate: func(in *app.SignUpInput) { in.OwnerName = "" }, wantErr: domain.ErrInvalidUserName},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newEnv(t)
				in := signUpInput()
				tt.mutate(&in)

				_, err := e.svc.SignUp(t.Context(), in)

				require.ErrorIs(t, err, tt.wantErr)
				require.Empty(t, e.tenants.byID)
				require.Empty(t, e.users.byID)
				require.Empty(t, e.tokens.byID)
			})
		}
	})
}

func TestLogin(t *testing.T) {
	login := func(email, password string) app.LoginInput {
		return app.LoginInput{Email: email, Password: password, ClientIP: "203.0.113.7"}
	}

	t.Run("inicia sesión con credenciales correctas y emite access y refresh token", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)

		session, err := e.svc.Login(t.Context(), login("  Caja@ElTornillo.co ", validPassword))

		require.NoError(t, err)
		require.Equal(t, user.ID(), session.Profile.User.ID())
		require.Equal(t, tenant.ID(), session.Profile.Tenant.ID())
		require.Equal(t, "access:"+tenant.ID().String()+":"+user.ID().String()+":cashier", session.AccessToken)
		requireUsableRefresh(t, e, session.RefreshToken, user.ID())
	})

	t.Run("contraseña errónea devuelve ErrInvalidCredentials", func(t *testing.T) {
		e := newEnv(t)
		e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)

		_, err := e.svc.Login(t.Context(), login("caja@eltornillo.co", "otra-clave-123"))

		require.ErrorIs(t, err, domain.ErrInvalidCredentials)
		require.Empty(t, e.tokens.byID)
	})

	t.Run("email inexistente devuelve el mismo error y también verifica un hash", func(t *testing.T) {
		e := newEnv(t)
		e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)

		_, err := e.svc.Login(t.Context(), login("nadie@eltornillo.co", validPassword))

		require.ErrorIs(t, err, domain.ErrInvalidCredentials)
		require.Equal(t, 1, e.hasher.Verifies(), "an unknown email costs one hash verification, like a known one")
	})

	t.Run("email mal formado devuelve ErrInvalidCredentials y también verifica un hash", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.Login(t.Context(), login("no-es-un-email", validPassword))

		require.ErrorIs(t, err, domain.ErrInvalidCredentials)
		require.Equal(t, 1, e.hasher.Verifies())
	})

	t.Run("usuario inactivo devuelve ErrInvalidCredentials aunque la contraseña sea correcta", func(t *testing.T) {
		e := newEnv(t)
		_, user := e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)
		user.Deactivate()
		require.NoError(t, e.users.Update(t.Context(), user))

		_, err := e.svc.Login(t.Context(), login("caja@eltornillo.co", validPassword))

		require.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("tenant suspendido devuelve ErrTenantSuspended solo con la contraseña correcta", func(t *testing.T) {
		e := newEnv(t)
		tenant, _ := e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)
		tenant.Suspend()
		require.NoError(t, e.tenants.Update(t.Context(), tenant))

		_, err := e.svc.Login(t.Context(), login("caja@eltornillo.co", "otra-clave-123"))
		require.ErrorIs(t, err, domain.ErrInvalidCredentials)

		_, err = e.svc.Login(t.Context(), login("caja@eltornillo.co", validPassword))
		require.ErrorIs(t, err, domain.ErrTenantSuspended)
		require.Empty(t, e.tokens.byID)
	})

	t.Run("el email en dos tenants con la misma contraseña pide elegir tenant", func(t *testing.T) {
		e := newEnv(t)
		tenantA, _ := e.seed(t, "ana@gmail.com", validPassword, domain.RoleOwner)
		tenantB, _ := e.seed(t, "ana@gmail.com", validPassword, domain.RoleCashier)

		_, err := e.svc.Login(t.Context(), login("ana@gmail.com", validPassword))

		require.ErrorIs(t, err, domain.ErrTenantSelectionRequired)
		var selection *app.TenantSelectionError
		require.True(t, errors.As(err, &selection))
		ids := []uuid.UUID{}
		for _, opt := range selection.Tenants {
			ids = append(ids, opt.ID)
			require.Equal(t, "Ferretería El Tornillo", opt.Name)
		}
		require.ElementsMatch(t, []uuid.UUID{tenantA.ID(), tenantB.ID()}, ids)
		require.Empty(t, e.tokens.byID)
	})

	t.Run("con tenant_id inicia sesión en ese tenant", func(t *testing.T) {
		e := newEnv(t)
		e.seed(t, "ana@gmail.com", validPassword, domain.RoleOwner)
		tenantB, userB := e.seed(t, "ana@gmail.com", validPassword, domain.RoleCashier)

		in := login("ana@gmail.com", validPassword)
		in.TenantID = tenantB.ID()
		session, err := e.svc.Login(t.Context(), in)

		require.NoError(t, err)
		require.Equal(t, userB.ID(), session.Profile.User.ID())
		require.Equal(t, tenantB.ID(), session.Profile.Tenant.ID())
	})

	t.Run("el email en dos tenants con contraseñas distintas inicia sesión en el que coincide", func(t *testing.T) {
		e := newEnv(t)
		e.seed(t, "ana@gmail.com", "clave-del-tenant-a", domain.RoleOwner)
		tenantB, userB := e.seed(t, "ana@gmail.com", validPassword, domain.RoleCashier)

		session, err := e.svc.Login(t.Context(), login("ana@gmail.com", validPassword))

		require.NoError(t, err)
		require.Equal(t, userB.ID(), session.Profile.User.ID())
		require.Equal(t, tenantB.ID(), session.Profile.Tenant.ID())
	})

	t.Run("un tenant_id que no corresponde a las credenciales devuelve ErrInvalidCredentials", func(t *testing.T) {
		e := newEnv(t)
		e.seed(t, "ana@gmail.com", validPassword, domain.RoleOwner)
		other, _ := e.seed(t, "otra@gmail.com", validPassword, domain.RoleOwner)

		in := login("ana@gmail.com", validPassword)
		in.TenantID = other.ID()
		_, err := e.svc.Login(t.Context(), in)

		require.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("supera el límite por IP devuelve ErrTooManyAttempts sin verificar la contraseña", func(t *testing.T) {
		e := newEnv(t)
		e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)
		e.ipLimiter.denied["203.0.113.7"] = true

		_, err := e.svc.Login(t.Context(), login("caja@eltornillo.co", validPassword))

		require.ErrorIs(t, err, domain.ErrTooManyAttempts)
		var limited *app.RateLimitError
		require.True(t, errors.As(err, &limited))
		require.Equal(t, time.Minute, limited.RetryAfter)
		require.Zero(t, e.hasher.Verifies())
	})

	t.Run("supera el límite por email devuelve ErrTooManyAttempts", func(t *testing.T) {
		e := newEnv(t)
		e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)
		e.emailLimiter.denied["caja@eltornillo.co"] = true

		_, err := e.svc.Login(t.Context(), login(" CAJA@eltornillo.co", validPassword))

		require.ErrorIs(t, err, domain.ErrTooManyAttempts)
		require.Zero(t, e.hasher.Verifies())
	})

	t.Run("registra cuadra.identity.logins con result success y failure", func(t *testing.T) {
		e := newEnv(t)
		e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)

		_, err := e.svc.Login(t.Context(), login("caja@eltornillo.co", validPassword))
		require.NoError(t, err)
		_, err = e.svc.Login(t.Context(), login("caja@eltornillo.co", "otra-clave-123"))
		require.Error(t, err)
		_, err = e.svc.Login(t.Context(), login("nadie@eltornillo.co", validPassword))
		require.Error(t, err)

		require.EqualValues(t, 1, e.loginCount(t, "success"))
		require.EqualValues(t, 2, e.loginCount(t, "failure"))
	})

	t.Run("el span identity.Login lleva ids y nunca el email ni la contraseña", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)

		_, err := e.svc.Login(t.Context(), login("caja@eltornillo.co", validPassword))
		require.NoError(t, err)

		span := e.span(t, "identity.Login")
		require.Contains(t, span.Attributes(), attribute.String("tenant.id", tenant.ID().String()))
		require.Contains(t, span.Attributes(), attribute.String("user.id", user.ID().String()))
		for _, kv := range span.Attributes() {
			require.NotContains(t, kv.Value.String(), "caja@eltornillo.co")
			require.NotContains(t, kv.Value.String(), validPassword)
		}
	})

	t.Run("un login fallido marca el span con error y su motivo", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.Login(t.Context(), login("nadie@eltornillo.co", validPassword))
		require.Error(t, err)

		span := e.span(t, "identity.Login")
		require.Equal(t, codes.Error, span.Status().Code)
		require.Contains(t, span.Attributes(), attribute.String("auth.failure_reason", "invalid_credentials"))
	})
}

func TestRefresh(t *testing.T) {
	// loggedIn signs a cashier in and returns the session.
	loggedIn := func(t *testing.T, e *env) app.Session {
		t.Helper()
		e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)
		session, err := e.svc.Login(t.Context(), app.LoginInput{Email: "caja@eltornillo.co", Password: validPassword, ClientIP: "203.0.113.7"})
		require.NoError(t, err)
		return session
	}
	familyOf := func(t *testing.T, e *env, raw string) uuid.UUID {
		t.Helper()
		tenantID, hash, err := domain.ParseRefreshToken(raw)
		require.NoError(t, err)
		stored, err := e.tokens.GetByHash(t.Context(), tenantID, hash)
		require.NoError(t, err)
		return stored.FamilyID()
	}

	t.Run("rota el token: el nuevo sirve y el anterior queda revocado", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		e.clock.Advance(time.Hour)

		second, err := e.svc.Refresh(t.Context(), first.RefreshToken)

		require.NoError(t, err)
		require.NotEqual(t, first.RefreshToken, second.RefreshToken)
		require.Equal(t, first.Profile.User.ID(), second.Profile.User.ID())
		require.Equal(t, e.clock.Now().Add(15*time.Minute), second.AccessExpiresAt)
		require.Equal(t, e.clock.Now().Add(domain.RefreshTokenTTL), second.RefreshExpiresAt)
		requireUsableRefresh(t, e, second.RefreshToken, first.Profile.User.ID())
		require.Equal(t, familyOf(t, e, first.RefreshToken), familyOf(t, e, second.RefreshToken))
		e.span(t, "identity.RefreshToken")
	})

	t.Run("un refresh token reutilizado invalida toda la familia", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		second, err := e.svc.Refresh(t.Context(), first.RefreshToken)
		require.NoError(t, err)

		// The first token is presented again: someone else has a copy.
		_, err = e.svc.Refresh(t.Context(), first.RefreshToken)
		require.ErrorIs(t, err, domain.ErrRefreshTokenReused)

		for _, token := range e.tokens.family(familyOf(t, e, first.RefreshToken)) {
			require.True(t, token.IsRevoked(), "every token of the family is revoked")
		}
		_, err = e.svc.Refresh(t.Context(), second.RefreshToken)
		require.ErrorIs(t, err, domain.ErrRefreshTokenReused, "the legitimate successor stops working too")
		require.EqualValues(t, 1, e.counter(t, "cuadra.identity.refresh_reuse_detected", "", ""),
			"counted once: when the reuse revoked live tokens")
	})

	t.Run("una rotación concurrente perdida se trata como reutilización", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		e.tokens.beforeRotate = func() {
			e.tokens.beforeRotate = nil
			_, err := e.svc.Refresh(t.Context(), first.RefreshToken)
			require.NoError(t, err)
		}

		_, err := e.svc.Refresh(t.Context(), first.RefreshToken)

		require.ErrorIs(t, err, domain.ErrRefreshTokenReused)
		for _, token := range e.tokens.family(familyOf(t, e, first.RefreshToken)) {
			require.True(t, token.IsRevoked())
		}
	})

	t.Run("token vencido es rechazado", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		e.clock.Advance(domain.RefreshTokenTTL)

		_, err := e.svc.Refresh(t.Context(), first.RefreshToken)

		require.ErrorIs(t, err, domain.ErrRefreshTokenExpired)
	})

	t.Run("token mal formado o desconocido es rechazado", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		tenantID, _, err := domain.ParseRefreshToken(first.RefreshToken)
		require.NoError(t, err)
		_, unknown, err := domain.NewRefreshToken(tenantID, uuid.Must(uuid.NewV7()), e.clock.Now())
		require.NoError(t, err)

		for _, raw := range []string{"", "basura", unknown} {
			_, err := e.svc.Refresh(t.Context(), raw)
			require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
		}
	})

	t.Run("usuario desactivado no puede refrescar y su familia queda revocada", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		user := first.Profile.User
		user.Deactivate()
		require.NoError(t, e.users.Update(t.Context(), user))

		_, err := e.svc.Refresh(t.Context(), first.RefreshToken)

		require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
		for _, token := range e.tokens.family(familyOf(t, e, first.RefreshToken)) {
			require.True(t, token.IsRevoked())
		}
	})

	t.Run("tenant suspendido no puede refrescar", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		tenant := first.Profile.Tenant
		tenant.Suspend()
		require.NoError(t, e.tenants.Update(t.Context(), tenant))

		_, err := e.svc.Refresh(t.Context(), first.RefreshToken)

		require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
	})

	t.Run("el refresh refleja el rol actual del usuario", func(t *testing.T) {
		e := newEnv(t)
		first := loggedIn(t, e)
		user := first.Profile.User
		require.NoError(t, user.ChangeRole(domain.RoleAdmin))
		require.NoError(t, e.users.Update(t.Context(), user))

		second, err := e.svc.Refresh(t.Context(), first.RefreshToken)

		require.NoError(t, err)
		require.Contains(t, second.AccessToken, ":admin")
	})
}

func TestLogout(t *testing.T) {
	t.Run("revoca la familia y el token ya no sirve", func(t *testing.T) {
		e := newEnv(t)
		e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)
		session, err := e.svc.Login(t.Context(), app.LoginInput{Email: "caja@eltornillo.co", Password: validPassword})
		require.NoError(t, err)

		require.NoError(t, e.svc.Logout(t.Context(), session.RefreshToken))

		_, err = e.svc.Refresh(t.Context(), session.RefreshToken)
		require.Error(t, err)
		require.Zero(t, e.counter(t, "cuadra.identity.refresh_reuse_detected", "", ""), "a logged-out token is not a theft signal")
		e.span(t, "identity.Logout")
	})

	t.Run("con un token desconocido o mal formado no falla", func(t *testing.T) {
		e := newEnv(t)
		_, unknown, err := domain.NewRefreshToken(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), e.clock.Now())
		require.NoError(t, err)

		for _, raw := range []string{"", "basura", unknown} {
			require.NoError(t, e.svc.Logout(t.Context(), raw))
		}
	})
}

func TestMe(t *testing.T) {
	t.Run("devuelve el usuario y su tenant", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)

		profile, err := e.svc.Me(t.Context(), tenant.ID(), user.ID())

		require.NoError(t, err)
		require.Equal(t, user.ID(), profile.User.ID())
		require.Equal(t, tenant.ID(), profile.Tenant.ID())
		e.span(t, "identity.GetMe")
	})

	t.Run("un usuario de otro tenant no se encuentra", func(t *testing.T) {
		e := newEnv(t)
		_, user := e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)
		other, _ := e.seed(t, "otra@eltornillo.co", validPassword, domain.RoleCashier)

		_, err := e.svc.Me(t.Context(), other.ID(), user.ID())

		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("usuario desactivado devuelve ErrUserNotFound", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "caja@eltornillo.co", validPassword, domain.RoleCashier)
		user.Deactivate()
		require.NoError(t, e.users.Update(t.Context(), user))

		_, err := e.svc.Me(t.Context(), tenant.ID(), user.ID())

		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("lista los módulos activos del tenant", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "bodega@eltornillo.co", validPassword, domain.RoleWarehouse)
		other, _ := e.seed(t, "bodega@modaurbana.co", validPassword, domain.RoleWarehouse)
		e.modules.activate(tenant.ID(), "variants", "lots_expiry")
		e.modules.activate(other.ID(), "work_orders")

		profile, err := e.svc.Me(t.Context(), tenant.ID(), user.ID())

		require.NoError(t, err)
		require.Equal(t, []string{"variants", "lots_expiry"}, profile.Modules)
	})

	t.Run("los permisos de un módulo inactivo no aparecen en /me", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "bodega@eltornillo.co", validPassword, domain.RoleWarehouse)

		profile, err := e.svc.Me(t.Context(), tenant.ID(), user.ID())
		require.NoError(t, err)
		require.Equal(t, []domain.Permission{
			domain.PermCatalogRead, domain.PermInventoryRead, domain.PermInventoryAdjust,
		}, profile.Permissions)

		e.modules.activate(tenant.ID(), "lots_expiry")
		profile, err = e.svc.Me(t.Context(), tenant.ID(), user.ID())
		require.NoError(t, err)
		require.Equal(t, []domain.Permission{
			domain.PermCatalogRead, domain.PermInventoryRead, domain.PermInventoryAdjust, "lots:manage",
		}, profile.Permissions)
	})
}

func TestUpdateTenant(t *testing.T) {
	rate := func(s string) *decimal.Decimal {
		d := decimal.RequireFromString(s)
		return &d
	}

	t.Run("actualiza el IVA por defecto de la empresa del usuario", func(t *testing.T) {
		tests := []struct{ template, rate string }{
			{template: "ferreteria", rate: "0.05"},
			{template: "drogueria", rate: "0"},
		}
		for _, tt := range tests {
			e := newEnv(t)
			in := signUpInput()
			in.BusinessTemplate = tt.template
			session, err := e.svc.SignUp(t.Context(), in)
			require.NoError(t, err)
			tenantID, userID := session.Profile.Tenant.ID(), session.Profile.User.ID()

			tenant, err := e.svc.UpdateTenant(t.Context(), tenantID, userID, app.TenantUpdate{DefaultTaxRate: rate(tt.rate)})

			require.NoError(t, err, tt.template)
			require.Equal(t, tt.rate, tenant.DefaultTaxRate().String(), tt.template)
			stored, err := e.tenants.GetByID(t.Context(), tenantID)
			require.NoError(t, err)
			require.Equal(t, tt.rate, stored.DefaultTaxRate().String(), tt.template)
			span := e.span(t, "identity.UpdateTenant")
			require.Contains(t, span.Attributes(), attribute.String("tenant.id", tenantID.String()))
			require.Contains(t, span.Attributes(), attribute.String("user.id", userID.String()))
		}
	})

	t.Run("sin cambios devuelve la empresa como está", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "ana@eltornillo.co", validPassword, domain.RoleOwner)

		got, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), user.ID(), app.TenantUpdate{})

		require.NoError(t, err)
		require.Equal(t, tenant.DefaultTaxRate().String(), got.DefaultTaxRate().String())
	})

	t.Run("un IVA inválido no se guarda", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "ana@eltornillo.co", validPassword, domain.RoleOwner)

		_, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), user.ID(), app.TenantUpdate{DefaultTaxRate: rate("1")})

		require.ErrorIs(t, err, domain.ErrInvalidDefaultTaxRate)
		stored, err := e.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.Equal(t, tenant.DefaultTaxRate().String(), stored.DefaultTaxRate().String())
		require.Equal(t, codes.Error, e.span(t, "identity.UpdateTenant").Status().Code)
	})

	t.Run("una empresa que no existe responde ErrTenantNotFound", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.UpdateTenant(t.Context(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), app.TenantUpdate{DefaultTaxRate: rate("0.19")})

		require.ErrorIs(t, err, domain.ErrTenantNotFound)
	})
}

func TestSessionAccess(t *testing.T) {
	t.Run("el inicio de sesión devuelve los módulos activos y los permisos del rol", func(t *testing.T) {
		e := newEnv(t)
		tenant, _ := e.seed(t, "caja@modaurbana.co", validPassword, domain.RoleCashier)
		e.modules.activate(tenant.ID(), "work_orders")

		session, err := e.svc.Login(t.Context(), app.LoginInput{Email: "caja@modaurbana.co", Password: validPassword, ClientIP: "203.0.113.7"})

		require.NoError(t, err)
		require.Equal(t, []string{"work_orders"}, session.Profile.Modules)
		require.Contains(t, session.Profile.Permissions, domain.Permission("work_orders:write"))
		require.NotContains(t, session.Profile.Permissions, domain.PermCatalogWrite)
	})

	t.Run("el registro devuelve el owner con todos los permisos del core y sin módulos", func(t *testing.T) {
		e := newEnv(t)

		session, err := e.svc.SignUp(t.Context(), signUpInput())

		require.NoError(t, err)
		require.Empty(t, session.Profile.Modules)
		var core []domain.Permission
		for _, g := range domain.CoreGrants() {
			core = append(core, g.Permission)
		}
		require.Equal(t, core, session.Profile.Permissions)
	})
}

func TestNewPermissionMatrix(t *testing.T) {
	t.Run("registra los permisos de los módulos junto a los del core", func(t *testing.T) {
		m, err := app.NewPermissionMatrix(testRegistry)

		require.NoError(t, err)
		require.True(t, m.Can(domain.RoleWarehouse, "lots:manage"))
		require.False(t, m.Can(domain.RoleCashier, "lots:manage"))
		require.True(t, m.Can(domain.RoleCashier, domain.PermSalesCreate))
	})

	t.Run("un módulo que declara un rol desconocido impide arrancar", func(t *testing.T) {
		_, err := app.NewPermissionMatrix(fakeRegistry{{Module: "lots_expiry", Permission: "lots:manage", Roles: []string{"root"}}})
		require.ErrorIs(t, err, domain.ErrInvalidRole)
	})

	t.Run("un módulo que repite un permiso del core impide arrancar", func(t *testing.T) {
		_, err := app.NewPermissionMatrix(fakeRegistry{{Module: "variants", Permission: "catalog:write", Roles: []string{"admin"}}})
		require.ErrorIs(t, err, domain.ErrDuplicatePermission)
	})

	t.Run("un módulo que declara un permiso mal escrito impide arrancar", func(t *testing.T) {
		_, err := app.NewPermissionMatrix(fakeRegistry{{Module: "lots_expiry", Permission: "Lots manage", Roles: []string{"admin"}}})
		require.ErrorIs(t, err, domain.ErrInvalidPermission)
	})
}
