package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
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

		require.EqualValues(t, 1, e.counter(t, "cuadra.auth.signups", "", ""))
		span := e.span(t, "identity.SignUp")
		require.Contains(t, span.Attributes(), attribute.String("tenant.id", tenant.ID().String()))
		require.Contains(t, span.Attributes(), attribute.String("user.id", user.ID().String()))
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

	t.Run("registra cuadra.auth.logins con result success y failure", func(t *testing.T) {
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
		require.EqualValues(t, 1, e.counter(t, "cuadra.auth.refresh_reuse_detected", "", ""),
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
		require.Zero(t, e.counter(t, "cuadra.auth.refresh_reuse_detected", "", ""), "a logged-out token is not a theft signal")
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
}
