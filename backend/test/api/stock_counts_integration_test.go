//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStockCountFlow counts a location while it keeps moving stock, reviews
// the differences and approves the count, through the whole stack.
func TestStockCountFlow(t *testing.T) {
	f := newSalesFlow(t)
	var countPath string

	t.Run("solo owner o admin inician un conteo", func(t *testing.T) {
		body := `{"location_id":"` + f.locationID + `"}`
		rec := f.do(t, http.MethodPost, "/api/v1/stock-counts", body, f.warehouseToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

		rec = f.do(t, http.MethodPost, "/api/v1/stock-counts", body, f.ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"lines":1`)
		countPath = "/api/v1/stock-counts/" + responseID(t, rec)

		rec = f.do(t, http.MethodPost, "/api/v1/stock-counts", body, f.ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "stock_count_in_progress")
	})

	t.Run("el negocio sigue moviendo stock y el cajero cuenta sin ver lo esperado", func(t *testing.T) {
		rec := f.do(t, http.MethodPost, "/api/v1/inventory/adjustments",
			`{"variant_id":"`+f.variantID+`","location_id":"`+f.locationID+`","quantity":"-1","reason":"Bulto roto"}`, f.ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		rec = f.do(t, http.MethodPost, countPath+"/entries", `{"variant_id":"`+f.variantID+`","quantity":"7"}`, f.cashierToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.NotContains(t, rec.Body.String(), "expected")

		rec = f.do(t, http.MethodGet, countPath, "", f.cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"counted_lines":1`)
		require.NotContains(t, rec.Body.String(), "expected")

		rec = f.do(t, http.MethodGet, countPath+"/lines", "", f.cashierToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})

	t.Run("la diferencia descuenta lo que se movió antes de contar", func(t *testing.T) {
		rec := f.do(t, http.MethodGet, countPath+"/lines?status=difference", "", f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var page struct {
			Items []map[string]any
			Total int
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		require.Equal(t, 1, page.Total)
		require.Equal(t, "9", page.Items[0]["expected_quantity"])
		require.Equal(t, "7", page.Items[0]["counted_quantity"])
		require.Equal(t, "-2", page.Items[0]["difference"])
		require.Equal(t, "-50000.00", page.Items[0]["difference_value"])
		require.Equal(t, f.productID, page.Items[0]["product_id"])
	})

	t.Run("otra empresa no ve el conteo", func(t *testing.T) {
		otherToken, _ := f.signup(t, "otra@conteos.co")
		rec := f.do(t, http.MethodGet, countPath, "", otherToken)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "stock_count_not_found")
	})

	t.Run("aprobar ajusta el stock con referencia al conteo y queda en el reporte", func(t *testing.T) {
		rec := f.do(t, http.MethodPost, countPath+"/approve", "", f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"status":"approved"`)
		require.Contains(t, rec.Body.String(), `"accuracy":"0.0000"`)

		rec = f.do(t, http.MethodGet, "/api/v1/variants/"+f.variantID+"/kardex?limit=1", "", f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"reason":"Conteo físico"`)
		require.Contains(t, rec.Body.String(), `"kind":"stock_count"`)
		require.Contains(t, rec.Body.String(), `"balance_after":"7"`)

		rec = f.do(t, http.MethodGet, "/api/v1/stock-counts?status=approved", "", f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"total":1`)
		require.Contains(t, rec.Body.String(), `"net_difference_value":"-50000.00"`)

		rec = f.do(t, http.MethodPost, countPath+"/approve", "", f.ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "stock_count_not_open")
	})
}
