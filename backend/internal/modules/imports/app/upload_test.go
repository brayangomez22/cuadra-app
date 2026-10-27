package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

func TestUploadImport(t *testing.T) {
	tests := []struct {
		business string
		rows     []string
		products int
	}{
		{"almacén de ropa: una camiseta en dos tallas y un jean", []string{
			line("Camiseta básica", "CAM-S", "25000", "10", "S"),
			line("Camiseta básica", "CAM-M", "25000", "8", "M"),
			line("Jean clásico", "JEAN-L", "90000", "", "L"),
		}, 2},
		{"ferretería: tres productos simples", []string{
			line("Tubo PVC 1/2", "TUB-12", "10000", "120", ""),
			line("Cable 12 AWG", "CAB-12", "3500", "500.5", ""),
			line("Brocha 2 pulgadas", "BRO-2", "7500", "", ""),
		}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.business, func(t *testing.T) {
			t.Run("valida y guarda una importación sin errores, con su previsualización", func(t *testing.T) {
				e := newEnv(t)

				p := e.upload(t, file(tt.rows...))

				require.Equal(t, domain.StatusValidated, p.Import.Status())
				require.Empty(t, p.Errors)
				require.Equal(t, len(tt.rows), p.Import.Counts().Rows)
				require.Equal(t, tt.products, p.Import.Counts().Products)
				require.Len(t, p.Sample, tt.products)
				require.Equal(t, domain.StatusValidated, e.imports.stored(p.Import.ID()).Status)
				require.Empty(t, e.catalog.products, "validating stores nothing")
				require.EqualValues(t, 1, e.counter(t, "cuadra.imports.uploads", "result", "validated"))
				attrs := e.span(t, "imports.UploadImport")
				require.Equal(t, p.Import.ID().String(), attrs["import.id"].AsString())
			})
		})
	}

	t.Run("con filas inválidas la rechaza, reporta cada error y no importa nada", func(t *testing.T) {
		e := newEnv(t)

		p := e.upload(t, file(
			line("Tubo PVC 1/2", "TUB-12", "10000", "", ""),
			line("Cable 12 AWG", "CAB-12", "diez mil", "", ""),
			line("Brocha", "MALO-1", "7500", "", ""),
		))

		require.Equal(t, domain.StatusRejected, p.Import.Status())
		require.Equal(t, []domain.RowError{
			{Row: 3, Column: domain.ColumnPrice, Code: domain.CodeNotANumber},
			{Row: 4, Code: "invalid_sku", Message: "El SKU no es válido."},
		}, p.Errors)
		require.Equal(t, 2, p.Import.Counts().Errors)
		page, err := e.svc.ListImportErrors(t.Context(), e.actor, p.Import.ID(), 10, 0)
		require.NoError(t, err)
		require.Equal(t, 2, page.Total)
		require.EqualValues(t, 1, e.counter(t, "cuadra.imports.uploads", "result", "rejected"))
	})

	t.Run("un archivo sin filas queda rechazado con el error en la fila 1", func(t *testing.T) {
		e := newEnv(t)

		p := e.upload(t, file())

		require.Equal(t, domain.StatusRejected, p.Import.Status())
		require.Equal(t, []domain.RowError{{Row: 1, Code: domain.CodeEmptyFile}}, p.Errors)
	})

	t.Run("sin el módulo de variantes un producto en varias filas es un error", func(t *testing.T) {
		e := newEnv(t)
		e.modules.active["variants"] = false

		p := e.upload(t, []byte("producto|unidad|sku|precio\nCamiseta|und|CAM-S|1\nCamiseta|und|CAM-M|1"))

		require.Equal(t, []domain.RowError{{Row: 3, Column: domain.ColumnProduct, Code: domain.CodeVariantsInactive}}, p.Errors)
	})

	t.Run("rechaza una sede que no existe", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.UploadImport(t.Context(), e.actor, app.UploadInput{
			FileName: "productos.csv", Format: domain.FormatCSV, Data: file(productRows(1)...),
			LocationID: uuid.Must(uuid.NewV7()), Policy: domain.UpdateExisting,
		})
		require.ErrorIs(t, err, app.ErrLocationNotFound)
		require.Empty(t, e.imports.state.imports)
	})

	t.Run("rechaza un archivo de más de 10 MB", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.UploadImport(t.Context(), e.actor, app.UploadInput{
			FileName: "productos.csv", Format: domain.FormatCSV, Data: make([]byte, domain.MaxFileBytes+1),
			LocationID: e.location, Policy: domain.UpdateExisting,
		})
		require.ErrorIs(t, err, domain.ErrFileTooLarge)
	})
}

func TestImportReadsIsolatedByTenant(t *testing.T) {
	t.Run("aislamiento: otra empresa no ve, no confirma ni lista los errores de la importación", func(t *testing.T) {
		e := newEnv(t)
		p := e.upload(t, file(productRows(1)...))
		other := app.Actor{TenantID: uuid.Must(uuid.NewV7()), UserID: e.actor.UserID}

		_, err := e.svc.GetImport(t.Context(), other, p.Import.ID())
		require.ErrorIs(t, err, domain.ErrImportNotFound)
		_, err = e.svc.ConfirmImport(t.Context(), other, p.Import.ID())
		require.ErrorIs(t, err, domain.ErrImportNotFound)
		_, err = e.svc.ListImportErrors(t.Context(), other, p.Import.ID(), 10, 0)
		require.ErrorIs(t, err, domain.ErrImportNotFound)
	})
}

func TestConfirmImport(t *testing.T) {
	t.Run("encola una importación validada", func(t *testing.T) {
		e := newEnv(t)

		imp := e.confirmed(t, file(productRows(1)...))

		require.Equal(t, domain.StatusQueued, imp.Status())
		require.Equal(t, []uuid.UUID{imp.ID()}, e.imports.enqueue)
		attrs := e.span(t, "imports.ConfirmImport")
		require.Equal(t, imp.ID().String(), attrs["import.id"].AsString())
	})

	t.Run("no confirma una importación rechazada ni la encola", func(t *testing.T) {
		e := newEnv(t)
		p := e.upload(t, file(line("Cable", "CAB-12", "diez mil", "", "")))

		_, err := e.svc.ConfirmImport(t.Context(), e.actor, p.Import.ID())
		require.ErrorIs(t, err, domain.ErrImportNotValidated)
		require.Empty(t, e.imports.enqueue)
	})
}

func TestImportTemplate(t *testing.T) {
	t.Run("con variantes incluye una columna por eje", func(t *testing.T) {
		e := newEnv(t)

		data, err := e.svc.ImportTemplate(t.Context(), e.actor, domain.FormatXLSX)
		require.NoError(t, err)
		require.Contains(t, string(data), domain.AxisPrefix+"talla")
		e.span(t, "imports.ImportTemplate")
	})

	t.Run("sin variantes no incluye ejes", func(t *testing.T) {
		e := newEnv(t)
		e.modules.active["variants"] = false

		data, err := e.svc.ImportTemplate(t.Context(), e.actor, domain.FormatCSV)
		require.NoError(t, err)
		require.NotContains(t, string(data), domain.AxisPrefix)
	})
}
