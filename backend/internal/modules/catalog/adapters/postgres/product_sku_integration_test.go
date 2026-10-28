//go:build integration

package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProductRepositoryBySKU(t *testing.T) {
	f := newFixture(t)
	tenant := f.createTenant(t)
	tube := f.createProduct(t, tenant, "TUB-12", "Tubo PVC 1/2")
	shirt := f.createProduct(t, tenant, "CAM-S", "Camiseta básica")
	f.createProduct(t, tenant, "BRO-2", "Brocha 2 pulgadas")

	t.Run("devuelve los productos con esos SKU, sin importar mayúsculas ni espacios", func(t *testing.T) {
		found, err := f.products.BySKU(t.Context(), tenant, []string{" tub-12", "cam-s", "NO-EXISTE"})
		require.NoError(t, err)

		ids := []any{}
		for _, p := range found {
			ids = append(ids, p.ID())
			require.NotEmpty(t, p.Variants())
		}
		require.ElementsMatch(t, []any{tube.ID(), shirt.ID()}, ids)
	})

	t.Run("aislamiento: otra empresa no ve los productos por su SKU", func(t *testing.T) {
		found, err := f.products.BySKU(t.Context(), f.createTenant(t), []string{"TUB-12"})
		require.NoError(t, err)
		require.Empty(t, found)
	})
}
