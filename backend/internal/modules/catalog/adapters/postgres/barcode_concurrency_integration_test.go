//go:build integration

package postgres_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// TestBarcodeUniquenessUnderConcurrency writes, at the same time, the same
// barcode to a variant of one product and to an alternate unit of another:
// each table's unique key cannot see the other table, so only the check
// under the tenant's lock keeps one of the two out (P60).
func TestBarcodeUniquenessUnderConcurrency(t *testing.T) {
	f := newFixture(t)
	tenant := f.createTenant(t)
	box := f.createUnit(t, tenant, "caja", "Caja")

	for i := range 15 {
		code := fmt.Sprintf("77099%08d", i)
		variantSide := f.newProduct(t, tenant, fmt.Sprintf("VAR-%d", i), "Con código en la variante")
		require.NoError(t, variantSide.UpdateVariant(variantSide.DefaultVariant().ID(), domain.VariantDetails{
			SKU: fmt.Sprintf("VAR-%d", i), Barcode: code, Cost: dec("1"),
		}))
		unitSide := f.createProduct(t, tenant, fmt.Sprintf("UNI-%d", i), "Con código en la unidad")
		require.NoError(t, unitSide.SetVariantUnit(unitSide.DefaultVariant().ID(), domain.VariantUnitParams{
			Unit: box.Ref(), Factor: dec("12"), Price: dec("5500"), Barcode: code,
		}))

		start := make(chan struct{})
		results := make(chan error, 2)
		go func() { <-start; results <- f.products.Create(t.Context(), variantSide) }()
		go func() { <-start; results <- f.products.Update(t.Context(), unitSide) }()
		close(start)
		first, second := <-results, <-results

		ok := 0
		for _, err := range []error{first, second} {
			if err == nil {
				ok++
				continue
			}
			require.ErrorIs(t, err, domain.ErrBarcodeTaken)
		}
		require.Equal(t, 1, ok, "el código %s quedó en las dos tablas", code)
	}
}
