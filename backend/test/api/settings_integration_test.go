//go:build integration

package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBusinessSettings changes the business's contact data and its stock
// and sales rules, and checks they rule the next operations, through the
// whole stack.
func TestBusinessSettings(t *testing.T) {
	f := newSalesFlow(t)

	t.Run("el dueño registra la dirección y el teléfono, y /me los trae", func(t *testing.T) {
		rec := f.do(t, http.MethodPatch, "/api/v1/tenant", `{"address":"Calle 45 # 12-30","city":"Medellín","phone":"604 444 1234"}`, f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"phone":"604 444 1234"`)

		rec = f.do(t, http.MethodGet, "/api/v1/me", "", f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"tenant_address":"Calle 45 # 12-30"`)
		require.Contains(t, rec.Body.String(), `"tenant_city":"Medellín"`)
	})

	t.Run("una dirección demasiado larga responde 400 y un teléfono con letras 422", func(t *testing.T) {
		rec := f.do(t, http.MethodPatch, "/api/v1/tenant", `{"address":"`+strings.Repeat("a", 201)+`"}`, f.ownerToken)
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

		rec = f.do(t, http.MethodPatch, "/api/v1/tenant", `{"phone":"llamar al dueño"}`, f.ownerToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "invalid_tenant_phone")
	})

	t.Run("el cajero y el bodeguero no ven ni cambian las reglas", func(t *testing.T) {
		for _, token := range []string{f.cashierToken, f.warehouseToken} {
			for _, path := range []string{"/api/v1/inventory/settings", "/api/v1/sales/settings"} {
				require.Equal(t, http.StatusForbidden, f.do(t, http.MethodGet, path, "", token).Code, path)
				require.Equal(t, http.StatusForbidden, f.do(t, http.MethodPatch, path, `{"require_cash_session":false}`, token).Code, path)
			}
		}
	})

	t.Run("con stock negativo permitido un ajuste puede sacar más de lo que hay", func(t *testing.T) {
		adjust := `{"variant_id":"` + f.variantID + `","location_id":"` + f.locationID + `","quantity":"-50","reason":"Venta sin registrar"}`
		rec := f.do(t, http.MethodPost, "/api/v1/inventory/adjustments", adjust, f.ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

		rec = f.do(t, http.MethodPatch, "/api/v1/inventory/settings", `{"allow_negative_stock":true}`, f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"allow_negative_stock":true}`, rec.Body.String())

		rec = f.do(t, http.MethodPost, "/api/v1/inventory/adjustments", adjust, f.ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	})

	t.Run("el descuento máximo que fija el dueño es el tope del cajero", func(t *testing.T) {
		rec := f.do(t, http.MethodGet, "/api/v1/sales/settings", "", f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"max_discount":"0","require_cash_session":true}`, rec.Body.String())

		rec = f.do(t, http.MethodPatch, "/api/v1/sales/settings", `{"max_discount":"0.1","require_cash_session":false}`, f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		rec = f.do(t, http.MethodGet, "/api/v1/sales/discount-limit", "", f.cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"unlimited":false,"max":"0.1"}`, rec.Body.String())

		rec = f.do(t, http.MethodPatch, "/api/v1/sales/settings", `{"max_discount":"1.5"}`, f.ownerToken)
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})

	t.Run("otra empresa sigue con sus reglas y sin datos de contacto", func(t *testing.T) {
		otherToken, _ := f.signup(t, "otra@ajustes.co")
		rec := f.do(t, http.MethodGet, "/api/v1/sales/settings", "", otherToken)
		require.JSONEq(t, `{"max_discount":"0","require_cash_session":true}`, rec.Body.String())
		rec = f.do(t, http.MethodGet, "/api/v1/inventory/settings", "", otherToken)
		require.JSONEq(t, `{"allow_negative_stock":false}`, rec.Body.String())
		rec = f.do(t, http.MethodGet, "/api/v1/me", "", otherToken)
		require.Contains(t, rec.Body.String(), `"tenant_address":""`)
	})
}
