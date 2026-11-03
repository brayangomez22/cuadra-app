package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// reorderLine is a data row of a product with one variant and its minimum
// and maximum stock.
func reorderLine(product, sku, minimum, maximum string) string {
	return lineOf(map[string]string{
		domain.ColumnProduct: product, domain.ColumnUnit: "und", domain.ColumnSKU: sku, domain.ColumnPrice: "1000",
		domain.ColumnReorderMin: minimum, domain.ColumnReorderMax: maximum,
	})
}

// rule returns the reorder rule of sku at the import's location.
func (e *env) rule(t *testing.T, sku string) (app.ReorderEntry, bool) {
	t.Helper()
	r, ok := e.stock.rules[reorderKey{e.location, e.catalog.ids[sku]}]
	return r, ok
}

func TestRunImportReorder(t *testing.T) {
	t.Run("guarda el mínimo y el máximo de cada variante en la sede de la importación", func(t *testing.T) {
		e := newEnv(t)
		imp := e.confirmed(t, file(
			reorderLine("Tubo PVC 1/2", "TUB-12", "20.5", "100"),
			reorderLine("Camiseta básica", "CAM-S", "3", "12"),
			reorderLine("Jean clásico", "JEAN-L", "", ""),
		))

		require.NoError(t, e.svc.RunImport(t.Context(), e.actor.TenantID, imp.ID()))

		require.Len(t, e.stock.rules, 2)
		tube, ok := e.rule(t, "TUB-12")
		require.True(t, ok)
		require.True(t, dec("20.5").Equal(tube.Min))
		require.True(t, dec("100").Equal(tube.Max))
		_, ok = e.rule(t, "JEAN-L")
		require.False(t, ok, "a row without minimum and maximum leaves the rule alone")
	})

	t.Run("actualizar un producto existente también actualiza su regla", func(t *testing.T) {
		e := newEnv(t)
		first := e.confirmed(t, file(reorderLine("Tubo PVC 1/2", "TUB-12", "", "")))
		require.NoError(t, e.svc.RunImport(t.Context(), e.actor.TenantID, first.ID()))

		again := e.confirmed(t, file(reorderLine("Tubo PVC 1/2", "TUB-12", "10", "50")))
		require.NoError(t, e.svc.RunImport(t.Context(), e.actor.TenantID, again.ID()))

		tube, ok := e.rule(t, "TUB-12")
		require.True(t, ok)
		require.True(t, dec("50").Equal(tube.Max))
	})

	t.Run("con la política de omitir, un producto existente no cambia su regla", func(t *testing.T) {
		e := newEnv(t)
		e.catalog.products["CAM-S"] = "Camiseta"
		p, err := e.svc.UploadImport(t.Context(), e.actor, skipInput(e, file(reorderLine("Camiseta", "CAM-S", "1", "5"))))
		require.NoError(t, err)
		_, err = e.svc.ConfirmImport(t.Context(), e.actor, p.Import.ID())
		require.NoError(t, err)

		require.NoError(t, e.svc.RunImport(t.Context(), e.actor.TenantID, p.Import.ID()))

		require.Empty(t, e.stock.rules)
	})

	t.Run("un lote rechazado no deja reglas", func(t *testing.T) {
		e := newEnv(t)
		imp := e.confirmed(t, file(reorderLine("Producto A", "SKU-A", "1", "5"), reorderLine("Producto B", "SKU-B", "1", "5")))
		e.catalog.failSKU = "SKU-B"

		require.NoError(t, e.svc.RunImport(t.Context(), e.actor.TenantID, imp.ID()))

		require.Equal(t, domain.StatusFailed, e.imports.stored(imp.ID()).Status)
		require.Empty(t, e.stock.rules)
	})

	t.Run("un mínimo mayor que el máximo se reporta en la previsualización", func(t *testing.T) {
		e := newEnv(t)

		p := e.upload(t, file(reorderLine("Camiseta", "CAM-S", "9", "3")))

		require.Equal(t, domain.StatusRejected, p.Import.Status())
		require.Equal(t, domain.CodeReorderRange, p.Errors[0].Code)
		require.Equal(t, 2, p.Errors[0].Row)
	})
}
