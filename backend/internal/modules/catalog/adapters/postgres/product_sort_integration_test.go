//go:build integration

package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// withPricing sets the default variant's cost and price.
func withPricing(cost, price string) productOption {
	return func(p *domain.NewProductParams) { p.Variant.Cost, p.Variant.Price = dec(cost), dec(price) }
}

func TestProductSort(t *testing.T) {
	f := newFixture(t)
	tenant := f.createTenant(t)
	// A hardware store's drill and screw and a clothing store's shirt:
	// prices and margins in a different order than their names.
	f.createProduct(t, tenant, "TAL-01", "Taladro percutor", withPricing("360000", "400000")) // 10 %
	f.createProduct(t, tenant, "TOR-14", "Tornillo 1/4", withPricing("150", "500"))           // 70 %
	f.createProduct(t, tenant, "CAM-M", "Camiseta básica", withPricing("22500", "30000"))     // 25 %
	active := true
	list := func(t *testing.T, sort domain.ProductSort) []string {
		t.Helper()
		page, err := f.products.Search(t.Context(), tenant, domain.ProductFilter{Active: &active, Sort: sort, Limit: 20})
		require.NoError(t, err)
		return names(page)
	}

	t.Run("por precio, el más barato primero", func(t *testing.T) {
		require.Equal(t, []string{"Tornillo 1/4", "Camiseta básica", "Taladro percutor"}, list(t, domain.SortByPrice))
	})
	t.Run("por margen, el más bajo primero", func(t *testing.T) {
		require.Equal(t, []string{"Taladro percutor", "Camiseta básica", "Tornillo 1/4"}, list(t, domain.SortByMargin))
	})
	t.Run("por nombre, como siempre", func(t *testing.T) {
		require.Equal(t, []string{"Camiseta básica", "Taladro percutor", "Tornillo 1/4"}, list(t, domain.SortByName))
	})
}
