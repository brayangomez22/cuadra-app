//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// createUserAt stores a user of tenantID with role, created at.
func (f fixture) createUserAt(t *testing.T, tenantID uuid.UUID, email string, role domain.Role, at time.Time) *domain.User {
	t.Helper()
	e, err := domain.ParseEmail(email)
	require.NoError(t, err)
	user, err := domain.NewUser(tenantID, e, "Usuario Demo", fakeHash, role, at)
	require.NoError(t, err)
	require.NoError(t, f.users.Create(t.Context(), user))
	return user
}

func TestUserRepositoryFirstByRole(t *testing.T) {
	f := newFixture(t)

	t.Run("devuelve el usuario activo más antiguo del rol", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createUserAt(t, tenant.ID(), "dueno@demo.test", domain.RoleOwner, now)
		first := f.createUserAt(t, tenant.ID(), "cajero@demo.test", domain.RoleCashier, now.Add(time.Minute))
		f.createUserAt(t, tenant.ID(), "cajera@demo.test", domain.RoleCashier, now.Add(time.Hour))

		got, err := f.users.FirstByRole(t.Context(), tenant.ID(), domain.RoleCashier)
		require.NoError(t, err)
		requireSameUser(t, first, got)
	})

	t.Run("salta los usuarios inactivos", func(t *testing.T) {
		tenant := f.createTenant(t)
		inactive := f.createUserAt(t, tenant.ID(), "bodega@demo.test", domain.RoleWarehouse, now)
		inactive.Deactivate()
		require.NoError(t, f.users.Update(t.Context(), inactive))
		active := f.createUserAt(t, tenant.ID(), "bodega2@demo.test", domain.RoleWarehouse, now.Add(time.Minute))

		got, err := f.users.FirstByRole(t.Context(), tenant.ID(), domain.RoleWarehouse)
		require.NoError(t, err)
		requireSameUser(t, active, got)
	})

	t.Run("sin usuario del rol devuelve ErrUserNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createUserAt(t, tenant.ID(), "dueno@demo.test", domain.RoleOwner, now)

		_, err := f.users.FirstByRole(t.Context(), tenant.ID(), domain.RoleAdmin)
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("aislamiento: no ve los usuarios de otro tenant", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		f.createUserAt(t, a.ID(), "admin@demo.test", domain.RoleAdmin, now)

		_, err := f.users.FirstByRole(t.Context(), b.ID(), domain.RoleAdmin)
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}
