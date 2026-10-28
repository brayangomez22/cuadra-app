package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// skipInput uploads data as a CSV file that skips the existing SKUs.
func skipInput(e *env, data []byte) app.UploadInput {
	return app.UploadInput{
		FileName: "productos.csv", Format: domain.FormatCSV, Data: data,
		LocationID: e.location, Policy: domain.SkipExisting,
	}
}

func TestRunImport(t *testing.T) {
	t.Run("crea los productos por lotes y registra el stock inicial de las variantes nuevas", func(t *testing.T) {
		e := newEnv(t)
		imp := e.confirmed(t, file(
			line("Camiseta básica", "CAM-S", "25000", "10", "S"),
			line("Camiseta básica", "CAM-M", "25000", "8", "M"),
			line("Jean clásico", "JEAN-L", "90000", "", "L"),
			line("Tubo PVC 1/2", "TUB-12", "10000", "120.5", ""),
		))

		require.NoError(t, e.svc.RunImport(t.Context(), e.actor.TenantID, imp.ID()))

		stored := e.imports.stored(imp.ID())
		require.Equal(t, domain.StatusCompleted, stored.Status)
		require.Equal(t, 3, stored.Counts.Created)
		require.Equal(t, 2, stored.BatchesDone, "a product is never split between batches")
		require.Len(t, e.catalog.products, 4)
		require.Len(t, e.stock.entries, 3)
		require.True(t, dec("120.5").Equal(e.stock.entries[2].Quantity))
		require.True(t, dec("100").Equal(e.stock.entries[2].UnitCost))
		require.EqualValues(t, 3, e.counter(t, "cuadra.imports.products", "result", "created"))
		require.EqualValues(t, 1, e.counter(t, "cuadra.imports.runs", "result", "completed"))
		attrs := e.span(t, "imports.RunImport")
		require.Equal(t, imp.ID().String(), attrs["import.id"].AsString())
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
	})

	t.Run("un SKU existente actualiza en vez de duplicar y no repite el stock inicial", func(t *testing.T) {
		e := newEnv(t)
		rows := []string{line("Tubo PVC 1/2", "TUB-12", "10000", "120", "")}
		require.NoError(t, e.svc.RunImport(t.Context(), e.actor.TenantID, e.confirmed(t, file(rows...)).ID()))

		again := e.confirmed(t, file(line("Tubo PVC 1/2 pulgada", "TUB-12", "11000", "120", "")))
		require.NoError(t, e.svc.RunImport(t.Context(), e.actor.TenantID, again.ID()))

		require.Equal(t, 1, e.imports.stored(again.ID()).Counts.Updated)
		require.Len(t, e.catalog.products, 1)
		require.Equal(t, "Tubo PVC 1/2 pulgada", e.catalog.products["TUB-12"])
		require.Len(t, e.stock.entries, 1)
	})

	t.Run("con la política de omitir, un SKU existente se cuenta como omitido", func(t *testing.T) {
		e := newEnv(t)
		e.catalog.products["TUB-12"] = "Tubo PVC"
		p, err := e.svc.UploadImport(t.Context(), e.actor, skipInput(e, file(line("Tubo nuevo", "TUB-12", "1", "", ""))))
		require.NoError(t, err)
		_, err = e.svc.ConfirmImport(t.Context(), e.actor, p.Import.ID())
		require.NoError(t, err)

		require.NoError(t, e.svc.RunImport(t.Context(), e.actor.TenantID, p.Import.ID()))

		require.Equal(t, 1, e.imports.stored(p.Import.ID()).Counts.Skipped)
		require.Equal(t, "Tubo PVC", e.catalog.products["TUB-12"])
	})

	t.Run("si un lote falla lo deshace, conserva los anteriores y la deja fallida con el error", func(t *testing.T) {
		e := newEnv(t)
		imp := e.confirmed(t, file(productRows(4)...))
		// The catalog rejects the fourth product only when it is written (as
		// a barcode another product took after the validation would).
		e.catalog.failSKU = "SKU-3"

		require.NoError(t, e.svc.RunImport(t.Context(), e.actor.TenantID, imp.ID()), "a business failure is not retried")

		stored := e.imports.stored(imp.ID())
		require.Equal(t, domain.StatusFailed, stored.Status)
		require.Equal(t, 1, stored.BatchesDone)
		require.Equal(t, 2, stored.Counts.Created)
		require.Contains(t, e.catalog.products, "SKU-1")
		require.NotContains(t, e.catalog.products, "SKU-2", "the failed batch is rolled back")
		page, err := e.svc.ListImportErrors(t.Context(), e.actor, imp.ID(), 10, 0)
		require.NoError(t, err)
		require.Equal(t, 5, page.Items[0].Row)
		require.EqualValues(t, 1, e.counter(t, "cuadra.imports.runs", "result", "failed"))
	})

	t.Run("reintentar después de una falla de infraestructura no repite los lotes hechos", func(t *testing.T) {
		e := newEnv(t)
		imp := e.confirmed(t, file(productRows(6)...))
		e.catalog.failOnCall = 2

		require.ErrorIs(t, e.svc.RunImport(t.Context(), e.actor.TenantID, imp.ID()), errConnection)
		require.Equal(t, 1, e.imports.stored(imp.ID()).BatchesDone)

		require.NoError(t, e.svc.RunImport(t.Context(), e.actor.TenantID, imp.ID()))

		stored := e.imports.stored(imp.ID())
		require.Equal(t, domain.StatusCompleted, stored.Status)
		require.Equal(t, 6, stored.Counts.Created)
		require.Len(t, e.stock.entries, 6, "no stock entry is repeated")
	})

	t.Run("una importación terminada no se vuelve a procesar", func(t *testing.T) {
		e := newEnv(t)
		imp := e.confirmed(t, file(productRows(1)...))
		require.NoError(t, e.svc.RunImport(t.Context(), e.actor.TenantID, imp.ID()))

		require.NoError(t, e.svc.RunImport(t.Context(), e.actor.TenantID, imp.ID()))

		require.Len(t, e.stock.entries, 1)
		require.Equal(t, 1, e.imports.stored(imp.ID()).Counts.Created)
	})
}
