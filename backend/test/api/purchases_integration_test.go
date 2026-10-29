//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestPurchasesFlow registers suppliers and purchases through the whole
// stack: a purchase in an alternate unit enters the stock in the base unit,
// updates the average cost and references the purchase in the kardex; on
// credit, it leaves a balance to pay to the supplier.
func TestPurchasesFlow(t *testing.T) {
	handler, tokens := newAPI(t)
	do := func(t *testing.T, method, path, body, bearer string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Authorization", "Bearer "+bearer)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	var session struct {
		AccessToken string `json:"access_token"`
		User        struct {
			TenantID uuid.UUID `json:"tenant_id"`
		} `json:"user"`
	}
	rec := do(t, http.MethodPost, "/api/v1/auth/signup",
		`{"tenant_name":"Ferretería El Tornillo","nit":"900123456-8","name":"Ana Gómez","email":"ana@compras.co","password":"clave-segura-123","business_template":"ferreteria"}`, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
	owner := session.AccessToken
	warehouse, _, err := tokens.Issue(session.User.TenantID, uuid.Must(uuid.NewV7()), "warehouse")
	require.NoError(t, err)

	rec = do(t, http.MethodPost, "/api/v1/products",
		`{"sku":"tor-14","name":"Tornillo drywall 6x1","base_unit_id":"`+unitID(t, do, owner, "und")+
			`","cost":"400","price":"600","tax_rate":"0.19","attributes":{}}`, owner)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var product struct {
		ID       string
		Variants []struct{ ID string }
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &product))
	variant, box := product.Variants[0].ID, unitID(t, do, owner, "caja")
	rec = do(t, http.MethodPut, "/api/v1/products/"+product.ID+"/variants/"+variant+"/units/"+box, `{"factor":"100","price":"55000"}`, owner)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = do(t, http.MethodPost, "/api/v1/locations", `{"name":"Bodega"}`, owner)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	location := responseID(t, rec)
	rec = do(t, http.MethodPost, "/api/v1/inventory/receipts",
		`{"variant_id":"`+variant+`","location_id":"`+location+`","quantity":"100","unit_cost":"400"}`, owner)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var supplier string
	purchase := func(invoice, payment string) string {
		return `{"supplier_id":"` + supplier + `","location_id":"` + location + `","invoice_number":"` + invoice +
			`","issued_on":"2026-10-20","payment":"` + payment + `","lines":[{"variant_id":"` + variant +
			`","unit_id":"` + box + `","quantity":"2","unit_cost":"45000"}]}`
	}

	t.Run("el bodeguero no ve proveedores ni registra compras", func(t *testing.T) {
		require.Equal(t, http.StatusForbidden, do(t, http.MethodGet, "/api/v1/suppliers", "", warehouse).Code)
		require.Equal(t, http.StatusForbidden, do(t, http.MethodPost, "/api/v1/purchases", "{}", warehouse).Code)
	})

	t.Run("crea el proveedor con su NIT y su plazo", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/suppliers",
			`{"name":"Distribuidora del Valle","nit":"901.234.567-7","contact_name":"Laura Pérez","payment_term_days":30}`, owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		supplier = responseID(t, rec)
		require.Contains(t, rec.Body.String(), `"nit":"901234567-7"`)

		rec = do(t, http.MethodPost, "/api/v1/suppliers", `{"name":"Otro","nit":"9012345677"}`, owner)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "supplier_nit_taken")
	})

	t.Run("una compra a crédito en cajas entra en unidad base, cambia el costo promedio y queda por pagar", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/purchases", purchase("FE 1234", "credit"), owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var p struct {
			ID, Status, Total string
			DueOn             string `json:"due_on"`
			BalanceDue        string `json:"balance_due"`
			Lines             []struct {
				UnitCode     string `json:"unit_code"`
				Quantity     string
				BaseQuantity string `json:"base_quantity"`
				BaseUnitCost string `json:"base_unit_cost"`
			}
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
		require.Equal(t, "open", p.Status)
		require.Equal(t, "107100.00", p.Total)
		require.Equal(t, "107100.00", p.BalanceDue)
		require.Equal(t, "2026-11-19", p.DueOn)
		require.Equal(t, "caja", p.Lines[0].UnitCode)
		require.Equal(t, "200", p.Lines[0].BaseQuantity)
		require.Equal(t, "450.00", p.Lines[0].BaseUnitCost)

		rec = do(t, http.MethodGet, "/api/v1/locations/"+location+"/stock?variant_id="+variant, "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"quantity":"300"`)
		// (100 × 400 + 200 × 450) / 300
		require.Contains(t, rec.Body.String(), `"average_cost":"433.3333"`)

		rec = do(t, http.MethodGet, "/api/v1/variants/"+variant+"/kardex?location_id="+location, "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"reference":{"id":"`+p.ID+`","kind":"purchase"}`)

		rec = do(t, http.MethodGet, "/api/v1/suppliers/"+supplier, "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"balance_due":"107100.00"`)
	})

	t.Run("no se puede registrar dos veces la misma factura del proveedor, y no mueve el stock", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/purchases", purchase("fe-1234", "cash"), owner)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "purchase_invoice_taken")
		rec = do(t, http.MethodGet, "/api/v1/locations/"+location+"/stock?variant_id="+variant, "", owner)
		require.Contains(t, rec.Body.String(), `"quantity":"300"`)
	})

	t.Run("lista las compras abiertas del proveedor y lee la compra", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/purchases", purchase("FE 1235", "cash"), owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		rec = do(t, http.MethodGet, "/api/v1/purchases?status=open&supplier_id="+supplier, "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var page struct {
			Total int
			Items []struct {
				ID           string
				SupplierName string `json:"supplier_name"`
			}
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		require.Equal(t, 1, page.Total)
		require.Equal(t, "Distribuidora del Valle", page.Items[0].SupplierName)

		rec = do(t, http.MethodGet, "/api/v1/purchases/"+page.Items[0].ID, "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"invoice_number":"FE 1234"`)
	})

	t.Run("una unidad que la variante no tiene dice la línea", func(t *testing.T) {
		body := strings.Replace(purchase("FE 9", "cash"), box, unitID(t, do, owner, "m"), 1)
		rec := do(t, http.MethodPost, "/api/v1/purchases", body, owner)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"code":"unit_not_available"`)
		require.Contains(t, rec.Body.String(), `"line":"0"`)
	})
}
