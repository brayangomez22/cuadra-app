package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestAuthorizeAction(t *testing.T) {
	authorize := func(tenantID uuid.UUID, email, password string) app.AuthorizationInput {
		return app.AuthorizationInput{
			TenantID: tenantID, Email: email, Password: password, Permission: domain.PermReceivablesOverride,
		}
	}

	t.Run("autoriza con las credenciales de un administrador del tenant", func(t *testing.T) {
		e := newEnv(t)
		tenant, cashier := e.seed(t, "caja@negocio.co", validPassword, domain.RoleCashier)
		admin := e.seedUser(t, tenant.ID(), "admin@negocio.co", "clave-admin-123", domain.RoleAdmin)
		id, err := e.svc.AuthorizeAction(t.Context(), authorize(tenant.ID(), " Admin@Negocio.co ", "clave-admin-123"))
		require.NoError(t, err)
		require.Equal(t, admin.ID(), id)
		require.NotEqual(t, cashier.ID(), id)
		e.span(t, "identity.AuthorizeAction")
	})

	t.Run("el dueño también autoriza", func(t *testing.T) {
		e := newEnv(t)
		tenant, owner := e.seed(t, "duena@negocio.co", validPassword, domain.RoleOwner)
		id, err := e.svc.AuthorizeAction(t.Context(), authorize(tenant.ID(), "duena@negocio.co", validPassword))
		require.NoError(t, err)
		require.Equal(t, owner.ID(), id)
	})

	t.Run("rechaza a un usuario cuyo rol no tiene el permiso", func(t *testing.T) {
		e := newEnv(t)
		tenant, _ := e.seed(t, "caja@negocio.co", validPassword, domain.RoleCashier)
		_, err := e.svc.AuthorizeAction(t.Context(), authorize(tenant.ID(), "caja@negocio.co", validPassword))
		require.ErrorIs(t, err, domain.ErrNotAuthorized)
	})

	t.Run("rechaza una contraseña errada, un correo desconocido o un usuario inactivo", func(t *testing.T) {
		e := newEnv(t)
		tenant, _ := e.seed(t, "admin@negocio.co", validPassword, domain.RoleAdmin)
		inactive := e.seedUser(t, tenant.ID(), "antes@negocio.co", validPassword, domain.RoleAdmin)
		inactive.Deactivate()
		require.NoError(t, e.users.Update(t.Context(), inactive))
		for _, in := range []app.AuthorizationInput{
			authorize(tenant.ID(), "admin@negocio.co", "otra-clave-123"),
			authorize(tenant.ID(), "nadie@negocio.co", validPassword),
			authorize(tenant.ID(), "antes@negocio.co", validPassword),
		} {
			_, err := e.svc.AuthorizeAction(t.Context(), in)
			require.ErrorIs(t, err, domain.ErrInvalidCredentials, in.Email)
		}
	})

	t.Run("rechaza al administrador de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		other, _ := e.seed(t, "admin@otro.co", validPassword, domain.RoleAdmin)
		mine, _ := e.seed(t, "caja@negocio.co", validPassword, domain.RoleCashier)
		require.NotEqual(t, other.ID(), mine.ID())
		_, err := e.svc.AuthorizeAction(t.Context(), authorize(mine.ID(), "admin@otro.co", validPassword))
		require.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("limita los intentos por correo como el inicio de sesión", func(t *testing.T) {
		e := newEnv(t)
		tenant, _ := e.seed(t, "admin@negocio.co", validPassword, domain.RoleAdmin)
		e.emailLimiter.denied["admin@negocio.co"] = true
		_, err := e.svc.AuthorizeAction(t.Context(), authorize(tenant.ID(), "Admin@negocio.co", validPassword))
		require.ErrorIs(t, err, domain.ErrTooManyAttempts)
	})
}
