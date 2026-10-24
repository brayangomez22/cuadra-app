//go:build integration

package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestCatalogTenantIsolation(t *testing.T) {
	f := newFixture(t)
	tenantA, tenantB := f.createTenant(t), f.createTenant(t)
	categoryA := f.createCategory(t, tenantA, "Plomería", nil)
	productA := f.createProduct(t, tenantA, "TUB-12", "Tubo PVC presión 1/2 pulgada", withCategory(categoryA.ID()))

	t.Run("el tenant B no ve los productos ni las categorías del tenant A", func(t *testing.T) {
		_, err := f.products.GetByID(t.Context(), tenantB, productA.ID())
		require.ErrorIs(t, err, domain.ErrProductNotFound)
		_, err = f.categories.GetByID(t.Context(), tenantB, categoryA.ID())
		require.ErrorIs(t, err, domain.ErrCategoryNotFound)

		list, err := f.categories.List(t.Context(), tenantB)
		require.NoError(t, err)
		require.Empty(t, list)
		chain, err := f.categories.Ancestors(t.Context(), tenantB, categoryA.ID())
		require.NoError(t, err)
		require.Empty(t, chain)
	})

	t.Run("el tenant B no encuentra los productos del tenant A al buscar", func(t *testing.T) {
		for _, q := range []string{"", "tubo", "TUB-12"} {
			page, err := f.products.Search(t.Context(), tenantB, domain.ProductFilter{Query: q, Limit: 20})
			require.NoError(t, err)
			require.Empty(t, page.Items, q)
			require.Zero(t, page.Total, q)
		}
	})

	t.Run("el tenant B no edita ni borra datos del tenant A", func(t *testing.T) {
		// The same entities, but claiming to belong to tenant B.
		forged, err := domain.RehydrateProduct(domain.ProductSnapshot{
			ID: productA.ID(), TenantID: tenantB, Name: "Cambiado", BaseUnit: f.piece[tenantB],
			TaxRate: productA.TaxRate(), Active: true, CreatedAt: now,
			Variants: []domain.VariantSnapshot{{
				ID: productA.DefaultVariant().ID(), SKU: "HACK", Cost: dec("1"), Price: dec("1"),
				IsDefault: true, Active: true, CreatedAt: now,
			}},
		})
		require.NoError(t, err)
		require.ErrorIs(t, f.products.Update(t.Context(), forged), domain.ErrProductNotFound)

		forgedCategory, err := domain.RehydrateCategory(domain.CategorySnapshot{
			ID: categoryA.ID(), TenantID: tenantB, Name: "Cambiada", CreatedAt: now,
		})
		require.NoError(t, err)
		require.ErrorIs(t, f.categories.Update(t.Context(), forgedCategory), domain.ErrCategoryNotFound)
		require.ErrorIs(t, f.categories.Delete(t.Context(), tenantB, categoryA.ID()), domain.ErrCategoryNotFound)

		got, err := f.products.GetByID(t.Context(), tenantA, productA.ID())
		require.NoError(t, err)
		require.Equal(t, "TUB-12", got.DefaultVariant().SKU())
		require.Equal(t, "Tubo PVC presión 1/2 pulgada", got.Name())
		gotCategory, err := f.categories.GetByID(t.Context(), tenantA, categoryA.ID())
		require.NoError(t, err)
		require.Equal(t, "Plomería", gotCategory.Name())
	})
}
