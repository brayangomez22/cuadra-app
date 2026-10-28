//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestVariantsInCategories(t *testing.T) {
	f := newFixture(t)

	t.Run("incluye las variantes de las subcategorías, activas o no", func(t *testing.T) {
		tenant := f.createTenant(t)
		tools := f.createCategory(t, tenant, "Herramientas", nil)
		toolsID := tools.ID()
		hand := f.createCategory(t, tenant, "Herramienta manual", &toolsID)
		clothes := f.createCategory(t, tenant, "Camisetas", nil)
		hammer := f.createProduct(t, tenant, "MAR-1", "Martillo", withCategory(hand.ID()))
		saw := f.createProduct(t, tenant, "SIE-1", "Sierra", withCategory(toolsID))
		saw.Deactivate()
		require.NoError(t, f.products.Update(t.Context(), saw))
		f.createProduct(t, tenant, "CAM-M", "Camiseta básica", withCategory(clothes.ID()))
		f.createProduct(t, tenant, "SUELTO", "Sin categoría")

		got, err := f.products.VariantsInCategories(t.Context(), tenant, []uuid.UUID{toolsID})
		require.NoError(t, err)
		require.ElementsMatch(t, []uuid.UUID{hammer.Variants()[0].ID(), saw.Variants()[0].ID()}, got)
	})

	t.Run("una categoría que no existe o es de otro tenant es ErrCategoryNotFound", func(t *testing.T) {
		tenantA, tenantB := f.createTenant(t), f.createTenant(t)
		own := f.createCategory(t, tenantA, "Papelería", nil)
		other := f.createCategory(t, tenantB, "Papelería", nil)
		for _, id := range []uuid.UUID{uuid.Must(uuid.NewV7()), other.ID()} {
			_, err := f.products.VariantsInCategories(t.Context(), tenantA, []uuid.UUID{own.ID(), id})
			require.ErrorIs(t, err, domain.ErrCategoryNotFound)
		}
	})
}
