package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

func TestUploadImport(t *testing.T) {
	location := uuid.Must(uuid.NewV7())

	t.Run("pasa el archivo, su formato, la sede y la política al caso de uso", func(t *testing.T) {
		uc := &stubUseCases{preview: app.Preview{Import: anImport(t)}}

		rec := serve(t, uc, uploadRequest(t, "Inventario.XLSX", []byte("datos"),
			map[string]string{"location_id": location.String(), "on_existing_sku": "skip"}))

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, app.UploadInput{
			FileName: "Inventario.XLSX", Format: domain.FormatXLSX, Data: []byte("datos"),
			LocationID: location, Policy: domain.SkipExisting,
		}, uc.upload)
		require.Equal(t, owner.TenantID, uc.actor.TenantID)
	})

	t.Run("sin política actualiza los SKU existentes", func(t *testing.T) {
		uc := &stubUseCases{preview: app.Preview{Import: anImport(t)}}

		serve(t, uc, uploadRequest(t, "productos.csv", []byte("x"), map[string]string{"location_id": location.String()}))

		require.Equal(t, domain.FormatCSV, uc.upload.Format)
		require.Equal(t, domain.UpdateExisting, uc.upload.Policy)
	})

	t.Run("describe cada error de fila con su columna", func(t *testing.T) {
		uc := &stubUseCases{preview: app.Preview{Import: anImport(t), Errors: []domain.RowError{
			{Row: 154, Column: "precio", Code: domain.CodeNotANumber},
			{Row: 7, Code: "barcode_taken", Message: "Ya existe ese código de barras."},
		}}}

		rec := serve(t, uc, uploadRequest(t, "productos.csv", []byte("x"), map[string]string{"location_id": location.String()}))

		var body struct {
			Errors []struct{ Message string } `json:"errors"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, "Fila 154, columna «precio»: no es un número.", body.Errors[0].Message)
		require.Equal(t, "Fila 7: Ya existe ese código de barras.", body.Errors[1].Message)
	})

	t.Run("la muestra trae el precio con formato o, si no es un número, el texto de la celda", func(t *testing.T) {
		uc := &stubUseCases{preview: app.Preview{Import: anImport(t), Sample: []domain.Group{{
			Row: 2, Product: domain.ProductFields{Name: "Camiseta"},
			Variants: []domain.VariantLine{
				{Row: 2, VariantFields: domain.VariantFields{SKU: "CAM-S", Price: decimal.RequireFromString("25000")}},
				{Row: 3, VariantFields: domain.VariantFields{SKU: "CAM-M", PriceText: "veinte mil"}},
			},
		}}}}

		rec := serve(t, uc, uploadRequest(t, "productos.csv", []byte("x"), map[string]string{"location_id": location.String()}))

		var body struct {
			Sample []struct {
				Variants []struct{ Price string } `json:"variants"`
			} `json:"sample"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, "25000", body.Sample[0].Variants[0].Price)
		require.Equal(t, "veinte mil", body.Sample[0].Variants[1].Price)
	})

	t.Run("un archivo que no es .xlsx ni .csv responde 422 unsupported_format", func(t *testing.T) {
		rec := serve(t, &stubUseCases{}, uploadRequest(t, "productos.pdf", []byte("x"), map[string]string{"location_id": location.String()}))
		requireError(t, rec, http.StatusUnprocessableEntity, "unsupported_format")
	})

	t.Run("errores del caso de uso", func(t *testing.T) {
		tests := []struct {
			err    error
			status int
			code   string
		}{
			{domain.ErrFileTooLarge, http.StatusRequestEntityTooLarge, "file_too_large"},
			{domain.ErrUnreadableFile, http.StatusUnprocessableEntity, "unreadable_file"},
			{app.ErrLocationNotFound, http.StatusUnprocessableEntity, "location_not_found"},
		}
		for _, tt := range tests {
			t.Run(tt.code, func(t *testing.T) {
				rec := serve(t, &stubUseCases{err: tt.err},
					uploadRequest(t, "productos.csv", []byte("x"), map[string]string{"location_id": location.String()}))
				requireError(t, rec, tt.status, tt.code)
			})
		}
	})
}

func TestConfirmImport(t *testing.T) {
	t.Run("una importación sin validar responde 409 import_not_validated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/imports/"+uuid.Must(uuid.NewV7()).String()+"/confirm", nil)
		requireError(t, serve(t, &stubUseCases{err: domain.ErrImportNotValidated}, req), http.StatusConflict, "import_not_validated")
	})

	t.Run("una importación de otra empresa responde 404 import_not_found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/imports/"+uuid.Must(uuid.NewV7()).String(), nil)
		requireError(t, serve(t, &stubUseCases{err: domain.ErrImportNotFound}, req), http.StatusNotFound, "import_not_found")
	})
}

func TestGetImportTemplate(t *testing.T) {
	t.Run("descarga la plantilla con su nombre y su tipo", func(t *testing.T) {
		uc := &stubUseCases{template: []byte("producto;sku")}

		rec := serve(t, uc, httptest.NewRequest(http.MethodGet, "/api/v1/import-template?format=csv", nil))

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, domain.FormatCSV, uc.format)
		require.Equal(t, "text/csv; charset=utf-8", rec.Header().Get("Content-Type"))
		require.Equal(t, `attachment; filename="plantilla-productos.csv"`, rec.Header().Get("Content-Disposition"))
		require.Equal(t, "producto;sku", rec.Body.String())
	})

	t.Run("sin formato descarga el .xlsx", func(t *testing.T) {
		uc := &stubUseCases{template: []byte("xlsx")}
		serve(t, uc, httptest.NewRequest(http.MethodGet, "/api/v1/import-template", nil))
		require.Equal(t, domain.FormatXLSX, uc.format)
	})
}
