package app_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

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
		require.ErrorAs(t, err, &selection)
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
		require.ErrorAs(t, err, &limited)
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
