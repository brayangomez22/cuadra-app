//go:build integration

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLargePurchase registers a purchase with more lines than the catalog
// reads in one call (P107): the contract accepts up to 200.
func TestLargePurchase(t *testing.T) {
	handler, _ := newAPI(t)
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
	cases := []struct{ template, email, article string }{
		{"ferreteria", "ana@compra-grande.co", "Tornillo"},
		{"ropa_calzado", "luis@compra-grande.co", "Camiseta"},
	}
	for _, c := range cases {
		t.Run("registra una compra de 150 líneas: "+c.template, func(t *testing.T) {
			rec := do(t, http.MethodPost, "/api/v1/auth/signup", `{"tenant_name":"Negocio Grande","nit":"900123456-8",
				"name":"Ana Gómez","email":"`+c.email+`","password":"clave-segura-123","business_template":"`+c.template+`"}`, "")
			require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
			var session struct {
				AccessToken string `json:"access_token"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
			owner := session.AccessToken
			und := unitID(t, do, owner, "und")

			rec = do(t, http.MethodPost, "/api/v1/locations", `{"name":"Principal"}`, owner)
			require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
			location := responseID(t, rec)
			rec = do(t, http.MethodPost, "/api/v1/suppliers", `{"name":"Mayorista","nit":"901234567-7"}`, owner)
			require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
			supplier := responseID(t, rec)

			lines := make([]string, 0, 150)
			for i := range 150 {
				rec := do(t, http.MethodPost, "/api/v1/products", fmt.Sprintf(`{"sku":"ART-%03d","name":"%s %d",
					"base_unit_id":"%s","cost":"1000","price":"1500","tax_rate":"0.19","attributes":{}}`, i, c.article, i, und), owner)
				require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
				var product struct{ Variants []struct{ ID string } }
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &product))
				lines = append(lines, `{"variant_id":"`+product.Variants[0].ID+`","quantity":"2","unit_cost":"1000"}`)
			}

			rec = do(t, http.MethodPost, "/api/v1/purchases", `{"supplier_id":"`+supplier+`","location_id":"`+location+
				`","invoice_number":"FV-150","issued_on":"2026-10-20","payment":"cash","lines":[`+strings.Join(lines, ",")+`]}`, owner)
			require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
			var purchase struct {
				Total string
				Lines []struct {
					BaseQuantity string `json:"base_quantity"`
				}
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &purchase))
			require.Len(t, purchase.Lines, 150)
			// 150 × 2 × 1000 = 300000, plus 19 % IVA.
			require.Equal(t, "357000.00", purchase.Total)
		})
	}
}
