package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// staff seeds a tenant with an owner, an admin and a cashier.
type staff struct {
	tenant                 *domain.Tenant
	owner, admin, cashier  *domain.User
	ownerActor, adminActor app.Actor
}

func (e *env) seedStaff(t *testing.T) staff {
	t.Helper()
	tenant, owner := e.seed(t, "duena@negocio.co", validPassword, domain.RoleOwner)
	s := staff{
		tenant: tenant, owner: owner,
		admin:   e.seedUser(t, tenant.ID(), "admin@negocio.co", validPassword, domain.RoleAdmin),
		cashier: e.seedUser(t, tenant.ID(), "caja@negocio.co", validPassword, domain.RoleCashier),
	}
	s.ownerActor = app.Actor{TenantID: tenant.ID(), UserID: owner.ID()}
	s.adminActor = app.Actor{TenantID: tenant.ID(), UserID: s.admin.ID()}
	return s
}

func TestListUsers(t *testing.T) {
	e := newEnv(t)
	s := e.seedStaff(t)
	other, _ := e.seed(t, "otra@empresa.co", validPassword, domain.RoleOwner)
	require.NotEqual(t, other.ID(), s.tenant.ID())

	users, err := e.svc.ListUsers(t.Context(), s.adminActor)
	require.NoError(t, err)
	require.Len(t, users, 3, "only the tenant's users")
	e.span(t, "identity.ListUsers")
}

func TestChangeUserRole(t *testing.T) {
	t.Run("el administrador cambia el rol de un cajero y queda en la auditoría", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		u, err := e.svc.ChangeUserRole(t.Context(), s.adminActor, s.cashier.ID(), "warehouse")
		require.NoError(t, err)
		require.Equal(t, domain.RoleWarehouse, u.Role())

		stored, err := e.users.GetByID(t.Context(), s.tenant.ID(), s.cashier.ID())
		require.NoError(t, err)
		require.Equal(t, domain.RoleWarehouse, stored.Role())
		require.Equal(t, []app.AuditEntry{{
			Action: "user.role_changed", EntityID: s.cashier.ID(),
			Details: map[string]string{"old_role": "cashier", "new_role": "warehouse"},
		}}, e.audit.of("user.role_changed"))
		require.Equal(t, int64(1), e.counter(t, "cuadra.identity.user_changes", "action", "role_changed"))
	})

	t.Run("el dueño no puede quitarse su propio rol de owner si es el único", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		_, err := e.svc.ChangeUserRole(t.Context(), s.ownerActor, s.owner.ID(), "admin")
		require.ErrorIs(t, err, domain.ErrLastOwner)
		require.Empty(t, e.audit.of("user.role_changed"))
	})

	t.Run("un admin no puede nombrar dueños", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		_, err := e.svc.ChangeUserRole(t.Context(), s.adminActor, s.cashier.ID(), "owner")
		require.ErrorIs(t, err, domain.ErrOwnerOnly)
	})

	t.Run("un usuario de otra empresa no existe", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		_, stranger := e.seed(t, "otra@empresa.co", validPassword, domain.RoleCashier)
		_, err := e.svc.ChangeUserRole(t.Context(), s.ownerActor, stranger.ID(), "admin")
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("si la auditoría falla el rol no cambia", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		e.audit.fail = errAudit
		_, err := e.svc.ChangeUserRole(t.Context(), s.ownerActor, s.cashier.ID(), "admin")
		require.ErrorIs(t, err, errAudit)
	})
}

func TestDeactivateUser(t *testing.T) {
	t.Run("desactivar revoca los refresh tokens del usuario", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		_, err := e.svc.Login(t.Context(), app.LoginInput{Email: "caja@negocio.co", Password: validPassword})
		require.NoError(t, err)
		require.Equal(t, 1, e.tokens.live(s.cashier.ID()))

		u, err := e.svc.DeactivateUser(t.Context(), s.adminActor, s.cashier.ID())
		require.NoError(t, err)
		require.False(t, u.IsActive())
		require.Zero(t, e.tokens.live(s.cashier.ID()))
		require.Len(t, e.audit.of("user.deactivated"), 1)

		_, err = e.svc.Login(t.Context(), app.LoginInput{Email: "caja@negocio.co", Password: validPassword})
		require.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})

	t.Run("reactivar devuelve el acceso y queda en la auditoría", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		_, err := e.svc.DeactivateUser(t.Context(), s.ownerActor, s.admin.ID())
		require.NoError(t, err)
		u, err := e.svc.ActivateUser(t.Context(), s.ownerActor, s.admin.ID())
		require.NoError(t, err)
		require.True(t, u.IsActive())
		require.Len(t, e.audit.of("user.activated"), 1)
	})

	t.Run("nadie se desactiva a sí mismo", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		_, err := e.svc.DeactivateUser(t.Context(), s.adminActor, s.admin.ID())
		require.ErrorIs(t, err, domain.ErrSelfDeactivation)
	})
}

func TestCurrentRole(t *testing.T) {
	t.Run("devuelve el rol guardado, no el del token", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		_, err := e.svc.ChangeUserRole(t.Context(), s.ownerActor, s.cashier.ID(), "admin")
		require.NoError(t, err)
		role, err := e.svc.CurrentRole(t.Context(), s.tenant.ID(), s.cashier.ID())
		require.NoError(t, err)
		require.Equal(t, domain.RoleAdmin, role)
	})

	t.Run("un usuario desactivado no tiene sesión", func(t *testing.T) {
		e := newEnv(t)
		s := e.seedStaff(t)
		_, err := e.svc.DeactivateUser(t.Context(), s.ownerActor, s.cashier.ID())
		require.NoError(t, err)
		_, err = e.svc.CurrentRole(t.Context(), s.tenant.ID(), s.cashier.ID())
		require.ErrorIs(t, err, domain.ErrUserNotFound)
		require.Equal(t, int64(1), e.counter(t, "cuadra.identity.sessions_rejected", "reason", "inactive"))
	})
}
