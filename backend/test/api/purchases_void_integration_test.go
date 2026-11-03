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

// TestPurchaseVoidAndPayments voids purchases and pays suppliers through the
// whole stack: voiding takes out of the stock what the purchase put in and
// is refused once that stock is gone; a payment lowers what is owed, which
// the payables add up by supplier.
func TestPurchaseVoidAndPayments(t *testing.T) {
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
		`{"tenant_name":"Almacén La Moda","nit":"900123456-8","name":"Ana Gómez","email":"ana@moda.co","password":"clave-segura-123","business_template":"ropa_calzado"}`, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
	owner := session.AccessToken
	warehouse, _, err := tokens.Issue(session.User.TenantID, uuid.Must(uuid.NewV7()), "warehouse")
	require.NoError(t, err)

	rec = do(t, http.MethodPost, "/api/v1/products",
		`{"sku":"med-01","name":"Medias deportivas","base_unit_id":"`+unitID(t, do, owner, "par")+
			`","cost":"4000","price":"9000","tax_rate":"0.19","attributes":{}}`, owner)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var product struct{ Variants []struct{ ID string } }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &product))
	variant := product.Variants[0].ID
	rec = do(t, http.MethodPost, "/api/v1/locations", `{"name":"Bodega"}`, owner)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	location := responseID(t, rec)
	rec = do(t, http.MethodPost, "/api/v1/suppliers", `{"name":"Confecciones Andinas","nit":"901234567-7","payment_term_days":30}`, owner)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	supplier := responseID(t, rec)

	buy := func(t *testing.T, invoice, payment, qty string) string {
		t.Helper()
		rec := do(t, http.MethodPost, "/api/v1/purchases", `{"supplier_id":"`+supplier+`","location_id":"`+location+
			`","invoice_number":"`+invoice+`","issued_on":"2026-10-20","payment":"`+payment+`","lines":[{"variant_id":"`+
			variant+`","quantity":"`+qty+`","unit_cost":"4500"}]}`, owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		return responseID(t, rec)
	}
	stock := func(t *testing.T) string {
		t.Helper()
		rec := do(t, http.MethodGet, "/api/v1/locations/"+location+"/stock?variant_ids="+variant, "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var page struct{ Items []struct{ Quantity string } }
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		require.Len(t, page.Items, 1)
		return page.Items[0].Quantity
	}

	t.Run("anular saca el stock que entró y queda en el kardex", func(t *testing.T) {
		id := buy(t, "FE-1", "cash", "36")
		require.Equal(t, "36", stock(t))

		rec := do(t, http.MethodPost, "/api/v1/purchases/"+id+"/void", `{"reason":"Factura duplicada"}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"status":"voided"`)
		require.Contains(t, rec.Body.String(), `"void_reason":"Factura duplicada"`)
		require.Equal(t, "0", stock(t))

		rec = do(t, http.MethodGet, "/api/v1/variants/"+variant+"/kardex?location_id="+location, "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"type":"purchase_void"`)

		rec = do(t, http.MethodPost, "/api/v1/purchases/"+id+"/void", `{"reason":"Otra vez"}`, owner)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "purchase_already_voided")
		rec = do(t, http.MethodGet, "/api/v1/purchases?status=voided", "", owner)
		require.Contains(t, rec.Body.String(), id)
	})

	t.Run("si ya salió parte del stock no anula y la compra queda igual", func(t *testing.T) {
		id := buy(t, "FE-2", "cash", "10")
		rec := do(t, http.MethodPost, "/api/v1/inventory/adjustments",
			`{"variant_id":"`+variant+`","location_id":"`+location+`","quantity":"-4","reason":"Pérdida"}`, owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		rec = do(t, http.MethodPost, "/api/v1/purchases/"+id+"/void", `{"reason":"Error"}`, owner)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"code":"insufficient_stock"`)
		require.Contains(t, rec.Body.String(), `"line":"0"`)
		require.Equal(t, "6", stock(t))
		rec = do(t, http.MethodGet, "/api/v1/purchases/"+id, "", owner)
		require.Contains(t, rec.Body.String(), `"status":"paid"`)
	})

	t.Run("los pagos bajan el saldo, no lo dejan negativo e impiden anular", func(t *testing.T) {
		// 10 pares a 4.500 + IVA 19 % = 53.550.
		id := buy(t, "FE-3", "credit", "10")
		rec := do(t, http.MethodPost, "/api/v1/purchases/"+id+"/payments", `{"method":"transfer","amount":"50000"}`, owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"balance_after":"3550.00"`)

		rec = do(t, http.MethodPost, "/api/v1/purchases/"+id+"/payments", `{"method":"cash","amount":"3550.01"}`, owner)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "payment_exceeds_balance")

		rec = do(t, http.MethodPost, "/api/v1/purchases/"+id+"/void", `{"reason":"Error"}`, owner)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "purchase_has_payments")

		rec = do(t, http.MethodGet, "/api/v1/purchases/"+id+"/payments", "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"amount":"50000.00"`)

		rec = do(t, http.MethodGet, "/api/v1/payables", "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"supplier_name":"Confecciones Andinas"`)
		require.Contains(t, rec.Body.String(), `"balance":"3550.00"`)
	})

	t.Run("el bodeguero no anula, no paga ni ve lo que se debe", func(t *testing.T) {
		id := buy(t, "FE-4", "credit", "1")
		require.Equal(t, http.StatusForbidden, do(t, http.MethodPost, "/api/v1/purchases/"+id+"/void", `{"reason":"x"}`, warehouse).Code)
		require.Equal(t, http.StatusForbidden, do(t, http.MethodPost, "/api/v1/purchases/"+id+"/payments", `{"method":"cash","amount":"1"}`, warehouse).Code)
		require.Equal(t, http.StatusForbidden, do(t, http.MethodGet, "/api/v1/payables", "", warehouse).Code)
	})
}
