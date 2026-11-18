package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

var adminNow = time.Date(2026, 11, 18, 15, 0, 0, 0, time.UTC)

func aUser(t *testing.T, role domain.Role) *domain.User {
	t.Helper()
	email := mustEmail(t, uuid.Must(uuid.NewV7()).String()+"@negocio.co")
	u, err := domain.NewUser(uuid.Must(uuid.NewV7()), email, "Ana", fakeHash, role, adminNow)
	require.NoError(t, err)
	return u
}

func TestChangeRoleBy(t *testing.T) {
	tests := []struct {
		name         string
		actor        domain.Role
		target       domain.Role
		newRole      domain.Role
		self         bool
		activeOwners int
		wantErr      error
	}{
		{name: "el dueño cambia un cajero a administrador", actor: domain.RoleOwner, target: domain.RoleCashier, newRole: domain.RoleAdmin, activeOwners: 1},
		{name: "el administrador cambia un bodeguero a cajero", actor: domain.RoleAdmin, target: domain.RoleWarehouse, newRole: domain.RoleCashier, activeOwners: 1},
		{name: "el dueño nombra otro dueño", actor: domain.RoleOwner, target: domain.RoleAdmin, newRole: domain.RoleOwner, activeOwners: 1},
		{name: "un dueño deja de serlo si hay otro", actor: domain.RoleOwner, target: domain.RoleOwner, newRole: domain.RoleAdmin, self: true, activeOwners: 2},
		{name: "un admin no puede asignar el rol owner", actor: domain.RoleAdmin, target: domain.RoleCashier, newRole: domain.RoleOwner, activeOwners: 1, wantErr: domain.ErrOwnerOnly},
		{name: "un admin no puede cambiar el rol de un owner", actor: domain.RoleAdmin, target: domain.RoleOwner, newRole: domain.RoleAdmin, activeOwners: 2, wantErr: domain.ErrOwnerOnly},
		{name: "rechaza quitar el rol owner al último owner activo", actor: domain.RoleOwner, target: domain.RoleOwner, newRole: domain.RoleAdmin, self: true, activeOwners: 1, wantErr: domain.ErrLastOwner},
		{name: "rechaza un rol inválido", actor: domain.RoleOwner, target: domain.RoleCashier, newRole: domain.Role("root"), activeOwners: 1, wantErr: domain.ErrInvalidRole},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actor := aUser(t, tt.actor)
			target := actor
			if !tt.self {
				target = aUser(t, tt.target)
			}
			before := target.Role()
			err := target.ChangeRoleBy(actor, tt.newRole, tt.activeOwners)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, before, target.Role(), "the role must not change on error")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.newRole, target.Role())
		})
	}
}

func TestDeactivateBy(t *testing.T) {
	t.Run("el administrador desactiva un cajero y lo reactiva", func(t *testing.T) {
		admin, cashier := aUser(t, domain.RoleAdmin), aUser(t, domain.RoleCashier)
		require.NoError(t, cashier.DeactivateBy(admin, 1))
		require.False(t, cashier.IsActive())
		require.NoError(t, cashier.ActivateBy(admin))
		require.True(t, cashier.IsActive())
	})

	t.Run("rechaza que un usuario se desactive a sí mismo", func(t *testing.T) {
		owner := aUser(t, domain.RoleOwner)
		require.ErrorIs(t, owner.DeactivateBy(owner, 2), domain.ErrSelfDeactivation)
		require.True(t, owner.IsActive())
	})

	t.Run("un admin no puede desactivar ni reactivar a un owner", func(t *testing.T) {
		admin, owner := aUser(t, domain.RoleAdmin), aUser(t, domain.RoleOwner)
		require.ErrorIs(t, owner.DeactivateBy(admin, 2), domain.ErrOwnerOnly)
		owner.Deactivate()
		require.ErrorIs(t, owner.ActivateBy(admin), domain.ErrOwnerOnly)
	})

	t.Run("rechaza desactivar al último owner activo", func(t *testing.T) {
		owner, other := aUser(t, domain.RoleOwner), aUser(t, domain.RoleOwner)
		require.ErrorIs(t, other.DeactivateBy(owner, 1), domain.ErrLastOwner)
		require.NoError(t, other.DeactivateBy(owner, 2))
	})
}

func TestCheckManages(t *testing.T) {
	t.Run("el administrador gestiona a un bodeguero pero no a un dueño", func(t *testing.T) {
		admin := aUser(t, domain.RoleAdmin)
		require.NoError(t, domain.CheckManages(admin, aUser(t, domain.RoleWarehouse)))
		require.ErrorIs(t, domain.CheckManages(admin, aUser(t, domain.RoleOwner)), domain.ErrOwnerOnly)
	})

	t.Run("solo el dueño invita con rol owner", func(t *testing.T) {
		require.NoError(t, domain.CheckAssignable(domain.RoleOwner, domain.RoleOwner))
		require.NoError(t, domain.CheckAssignable(domain.RoleAdmin, domain.RoleCashier))
		require.ErrorIs(t, domain.CheckAssignable(domain.RoleAdmin, domain.RoleOwner), domain.ErrOwnerOnly)
	})
}

func TestChangePasswordHash(t *testing.T) {
	u := aUser(t, domain.RoleCashier)
	require.ErrorIs(t, u.SetPasswordHash(""), domain.ErrEmptyPasswordHash)
	require.NoError(t, u.SetPasswordHash("$argon2id$nuevo"))
	require.Equal(t, "$argon2id$nuevo", u.PasswordHash())
}
