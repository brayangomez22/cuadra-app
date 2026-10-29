//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

var _ domain.DemoTenantRepository = (*postgres.DemoTenantRepository)(nil)

// createTenantOf stores a tenant of template, marked demo or not, with a user.
func (f fixture) createTenantOf(t *testing.T, template string, demo bool) *domain.Tenant {
	t.Helper()
	nit, err := domain.ParseNIT("900123456-8")
	require.NoError(t, err)
	tenant, err := domain.NewTenant("Negocio Demo", nit, template, decimal.RequireFromString("0.19"), now)
	require.NoError(t, err)
	if demo {
		tenant.MarkDemo()
	}
	require.NoError(t, f.tenants.Create(t.Context(), tenant))
	f.createUser(t, tenant.ID(), "ana@negocio.test")
	return tenant
}

func TestDemoTenantRepository(t *testing.T) {
	f := newFixture(t)
	demo := postgres.NewDemoTenantRepository(f.db)

	t.Run("guarda y lee la marca de demostración", func(t *testing.T) {
		for _, template := range []string{"ferreteria", "ropa_calzado"} {
			tenant := f.createTenantOf(t, template, true)
			got, err := f.tenants.GetByID(t.Context(), tenant.ID())
			require.NoError(t, err)
			require.True(t, got.IsDemo())
		}
	})

	t.Run("encuentra solo los tenants demo de la plantilla", func(t *testing.T) {
		wanted := f.createTenantOf(t, "papeleria", true)
		f.createTenantOf(t, "papeleria", false)
		f.createTenantOf(t, "tienda", true)

		ids, err := demo.FindByTemplate(t.Context(), "papeleria")
		require.NoError(t, err)
		require.Equal(t, []uuid.UUID{wanted.ID()}, ids)
	})

	t.Run("borra el tenant demo con sus usuarios", func(t *testing.T) {
		tenant := f.createTenantOf(t, "drogueria", true)

		require.NoError(t, demo.Purge(t.Context(), tenant.ID()))
		_, err := f.tenants.GetByID(t.Context(), tenant.ID())
		require.ErrorIs(t, err, domain.ErrTenantNotFound)
		_, err = f.users.GetByEmail(t.Context(), tenant.ID(), mustEmail(t, "ana@negocio.test"))
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("aislamiento: no borra un tenant que no es demo ni toca a los demás", func(t *testing.T) {
		kept := f.createTenantOf(t, "generica", false)
		other := f.createTenantOf(t, "generica", true)

		require.ErrorIs(t, demo.Purge(t.Context(), kept.ID()), domain.ErrNotDemoTenant)
		require.NoError(t, demo.Purge(t.Context(), other.ID()))
		_, err := f.tenants.GetByID(t.Context(), kept.ID())
		require.NoError(t, err)
		_, err = f.users.GetByEmail(t.Context(), kept.ID(), mustEmail(t, "ana@negocio.test"))
		require.NoError(t, err)
		require.ErrorIs(t, demo.Purge(t.Context(), uuid.Must(uuid.NewV7())), domain.ErrTenantNotFound)
	})
}

func mustEmail(t *testing.T, s string) domain.Email {
	t.Helper()
	e, err := domain.ParseEmail(s)
	require.NoError(t, err)
	return e
}
