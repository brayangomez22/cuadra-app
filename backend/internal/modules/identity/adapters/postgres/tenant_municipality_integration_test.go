//go:build integration

package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestTenantRepositoryMunicipality(t *testing.T) {
	f := newFixture(t)

	withMunicipality := func(t *testing.T, code string) domain.Contact {
		t.Helper()
		c, err := domain.NewContact("Cra 50 # 40-20", "Medellín", "")
		require.NoError(t, err)
		c, err = c.WithMunicipality(code)
		require.NoError(t, err)
		return c
	}

	t.Run("guarda y borra el municipio DANE de la empresa", func(t *testing.T) {
		tenant := f.createTenant(t)
		tenant.SetContact(withMunicipality(t, "05001"))
		require.NoError(t, f.tenants.Update(t.Context(), tenant))

		got, err := f.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.Equal(t, "05001", got.Contact().MunicipalityCode())

		got.SetContact(withMunicipality(t, ""))
		require.NoError(t, f.tenants.Update(t.Context(), got))
		cleared, err := f.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.Empty(t, cleared.Contact().MunicipalityCode())
	})

	t.Run("rechaza un municipio que no está en el catálogo del DANE", func(t *testing.T) {
		tenant := f.createTenant(t)
		tenant.SetContact(withMunicipality(t, "99999"))
		require.ErrorIs(t, f.tenants.Update(t.Context(), tenant), domain.ErrInvalidMunicipality)
	})
}
