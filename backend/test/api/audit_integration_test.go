//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// auditPage is the body of GET /audit-events.
type auditPage struct {
	Items []struct {
		Action     string            `json:"action"`
		Role       string            `json:"role"`
		EntityType string            `json:"entity_type"`
		EntityID   string            `json:"entity_id"`
		LocationID string            `json:"location_id"`
		Details    map[string]string `json:"details"`
	} `json:"items"`
	Total int `json:"total"`
}

// TestAuditLog runs the audited operations through the whole stack: each
// one leaves its event in the same transaction, a failed one leaves none,
// and only the owner reads the log.
func TestAuditLog(t *testing.T) {
	f := newSalesFlow(t)
	do, owner, cashier := f.do, f.ownerToken, f.cashierToken

	events := func(t *testing.T, query string) auditPage {
		t.Helper()
		rec := do(t, http.MethodGet, "/api/v1/audit-events"+query, "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var page auditPage
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		return page
	}

	var saleID, sessionID string
	t.Run("una venta anulada y una caja con faltante quedan en el registro", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/cash-sessions", `{"location_id":"`+f.locationID+`","opening_float":"100000"}`, cashier)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		sessionID = responseID(t, rec)
		rec = do(t, http.MethodPost, "/api/v1/sales", f.saleBody("1", "35700"), cashier)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		saleID = responseID(t, rec)

		rec = do(t, http.MethodPost, "/api/v1/sales/"+saleID+"/void", `{"reason":"El cliente desistió"}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPost, "/api/v1/cash-sessions/"+sessionID+"/close",
			`{"counts":[{"denomination":"50000","quantity":1}]}`, cashier)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		voided := events(t, "?action=sale.voided")
		require.Equal(t, 1, voided.Total)
		require.Equal(t, "owner", voided.Items[0].Role)
		require.Equal(t, saleID, voided.Items[0].EntityID)
		require.Equal(t, f.locationID, voided.Items[0].LocationID)
		require.Equal(t, map[string]string{"total": "35700.00"}, voided.Items[0].Details)

		closed := events(t, "?action=cash.closed_with_difference")
		require.Equal(t, 1, closed.Total)
		require.Equal(t, "cashier", closed.Items[0].Role)
		require.Equal(t, map[string]string{"expected": "100000.00", "counted": "50000.00", "difference": "-50000.00"}, closed.Items[0].Details)
	})

	t.Run("una operación que falla no deja evento", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/sales/"+saleID+"/void", `{"reason":"Otra vez"}`, owner)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Equal(t, 1, events(t, "?action=sale.voided").Total)
	})

	t.Run("el cambio de precio, el ajuste, el módulo y el IVA quedan en el registro", func(t *testing.T) {
		rec := do(t, http.MethodPut, "/api/v1/products/"+f.productID+"/variants/"+f.variantID,
			`{"sku":"cem-50","cost":"25000","price":"31500","axis_values":{},"active":true}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPost, "/api/v1/inventory/adjustments",
			`{"variant_id":"`+f.variantID+`","location_id":"`+f.locationID+`","quantity":"-0.5","reason":"Bulto roto"}`, f.warehouseToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPut, "/api/v1/modules/variants", `{"active":true}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPatch, "/api/v1/tenant", `{"default_tax_rate":"0.05"}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		price := events(t, "?action=variant.price_changed&entity_id="+f.variantID)
		require.Equal(t, 1, price.Total)
		require.Equal(t, map[string]string{"old_price": "30000.00", "new_price": "31500.00"}, price.Items[0].Details)

		adjusted := events(t, "?action=stock.adjusted")
		require.Equal(t, 1, adjusted.Total)
		require.Equal(t, "warehouse", adjusted.Items[0].Role)
		require.Equal(t, map[string]string{"quantity": "-0.5", "source": "adjustment"}, adjusted.Items[0].Details)

		module := events(t, "?action=module.enabled")
		require.Equal(t, 1, module.Total, "la plantilla del registro no deja eventos")
		require.Equal(t, "module", module.Items[0].EntityType)
		require.Empty(t, module.Items[0].EntityID)
		require.Equal(t, map[string]string{"module": "variants"}, module.Items[0].Details)

		tax := events(t, "?action=tenant.settings_changed")
		require.Equal(t, 1, tax.Total)
		require.Equal(t, map[string]string{"setting": "default_tax_rate", "old": "0.19", "new": "0.05"}, tax.Items[0].Details)

		require.Equal(t, 2, events(t, "?action=user.invited").Total, "el cajero y el bodeguero del flujo entraron por invitación")
		require.Equal(t, 8, events(t, "").Total)
	})

	t.Run("solo el dueño consulta el registro", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/audit-events", "", cashier)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})

	t.Run("aislamiento: otra empresa no ve los eventos", func(t *testing.T) {
		other, _ := f.signup(t, "otro@calzado.co")
		rec := do(t, http.MethodGet, "/api/v1/audit-events", "", other)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"items":[],"total":0}`, rec.Body.String())
	})
}
