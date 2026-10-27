//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestReorderFlow defines a variant's minimum and maximum, sells it down to
// the minimum and reads the alert, through the whole stack.
func TestReorderFlow(t *testing.T) {
	f := newSalesFlow(t)
	rulePath := "/api/v1/locations/" + f.locationID + "/reorder-rules/" + f.variantID
	alertsPath := "/api/v1/inventory/reorder-alerts?location_id=" + f.locationID

	alerts := func(t *testing.T, bearer string) (total int, items []map[string]any) {
		t.Helper()
		rec := f.do(t, http.MethodGet, alertsPath, "", bearer)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var page struct {
			Items []map[string]any
			Total int
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		return page.Total, page.Items
	}

	t.Run("el cajero no define mínimos; el bodeguero sí", func(t *testing.T) {
		body := `{"min_quantity":"4","max_quantity":"12"}`
		rec := f.do(t, http.MethodPut, rulePath, body, f.cashierToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

		rec = f.do(t, http.MethodPut, rulePath, body, f.warehouseToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"min_quantity":"4"`)
	})

	t.Run("un mínimo mayor que el máximo responde 422", func(t *testing.T) {
		rec := f.do(t, http.MethodPut, rulePath, `{"min_quantity":"9","max_quantity":"3"}`, f.ownerToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "invalid_reorder_range")
	})

	t.Run("con stock sobre el mínimo no hay alerta; la existencia trae su regla", func(t *testing.T) {
		total, _ := alerts(t, f.cashierToken)
		require.Equal(t, 0, total)

		rec := f.do(t, http.MethodGet, "/api/v1/locations/"+f.locationID+"/stock", "", f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"reorder":{"max_quantity":"12","min_quantity":"4","needs_reorder":false}`)
	})

	t.Run("al bajar el stock hasta el mínimo aparece con lo que hay que pedir", func(t *testing.T) {
		rec := f.do(t, http.MethodPost, "/api/v1/inventory/adjustments",
			`{"variant_id":"`+f.variantID+`","location_id":"`+f.locationID+`","quantity":"-6","reason":"Mercancía averiada"}`, f.ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		total, items := alerts(t, f.cashierToken)
		require.Equal(t, 1, total)
		require.Equal(t, f.variantID, items[0]["variant_id"])
		require.Equal(t, f.productID, items[0]["product_id"])
		require.Equal(t, "4", items[0]["quantity"])
		require.Equal(t, "8", items[0]["suggested_quantity"])
	})

	t.Run("otra empresa no ve la alerta ni puede quitar la regla", func(t *testing.T) {
		otherToken, _ := f.signup(t, "otra@reposicion.co")
		rec := f.do(t, http.MethodGet, "/api/v1/inventory/reorder-alerts", "", otherToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"total":0`)

		rec = f.do(t, http.MethodDelete, rulePath, "", otherToken)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	})

	t.Run("al quitar la regla deja de aparecer", func(t *testing.T) {
		rec := f.do(t, http.MethodDelete, rulePath, "", f.ownerToken)
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		total, _ := alerts(t, f.ownerToken)
		require.Equal(t, 0, total)
	})
}
