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

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

// salesFlow is the whole stack TestSalesFlow sells through: a tenant with
// an owner, a cashier and a warehouse user, and one simple product with
// stock at a location.
type salesFlow struct {
	handler http.Handler
	tokens  *auth.JWT

	ownerToken, cashierToken, warehouseToken string
	tenantID                                 uuid.UUID
	productID, variantID, locationID         string
}

// newSalesFlow builds the modules, signs the tenant up and receives 10
// units of its product.
func newSalesFlow(t *testing.T) *salesFlow {
	t.Helper()
	f := &salesFlow{}
	f.handler, f.tokens = newAPI(t)

	var err error
	f.ownerToken, f.tenantID = f.signup(t, "ana@ventas.co")
	f.cashierToken, _, err = f.tokens.Issue(f.tenantID, uuid.Must(uuid.NewV7()), "cashier")
	require.NoError(t, err)
	f.warehouseToken, _, err = f.tokens.Issue(f.tenantID, uuid.Must(uuid.NewV7()), "warehouse")
	require.NoError(t, err)

	rec := f.do(t, http.MethodPost, "/api/v1/products",
		`{"sku":"cem-50","name":"Cemento gris 50kg","base_unit_id":"`+unitID(t, f.do, f.ownerToken, "bulto")+
			`","cost":"25000","price":"30000","tax_rate":"0.19","attributes":{}}`, f.ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	f.productID = responseID(t, rec)
	// A simple product sells its default variant, whose id is not the
	// product's.
	var created struct {
		Variants []struct{ ID string } `json:"variants"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	f.variantID = created.Variants[0].ID
	require.NotEqual(t, f.productID, f.variantID)
	rec = f.do(t, http.MethodPost, "/api/v1/locations", `{"name":"Mostrador"}`, f.ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	f.locationID = responseID(t, rec)
	rec = f.do(t, http.MethodPost, "/api/v1/inventory/receipts",
		`{"variant_id":"`+f.variantID+`","location_id":"`+f.locationID+`","quantity":"10","unit_cost":"25000"}`, f.ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	return f
}

func (f *salesFlow) do(t *testing.T, method, path, body, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

// signup registers a business and returns its owner's token and its id.
func (f *salesFlow) signup(t *testing.T, email string) (token string, tenantID uuid.UUID) {
	t.Helper()
	rec := f.do(t, http.MethodPost, "/api/v1/auth/signup",
		`{"tenant_name":"Ferretería El Tornillo","nit":"900123456-8","name":"Ana Gómez","email":"`+email+`","password":"clave-segura-123","business_template":"ferreteria"}`, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var session struct {
		AccessToken string `json:"access_token"`
		User        struct {
			TenantID uuid.UUID `json:"tenant_id"`
		} `json:"user"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
	return session.AccessToken, session.User.TenantID
}

// stockOf returns the variant's stock at the flow's location.
func (f *salesFlow) stockOf(t *testing.T, variantID string) string {
	t.Helper()
	rec := f.do(t, http.MethodGet, "/api/v1/locations/"+f.locationID+"/stock?variant_id="+variantID, "", f.ownerToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var page struct {
		Items []struct{ Quantity string }
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	require.Len(t, page.Items, 1)
	return page.Items[0].Quantity
}

// saleBody is a sale of qty of the product, paid with cash.
func (f *salesFlow) saleBody(qty, cash string) string {
	return `{"location_id":"` + f.locationID + `","lines":[{"variant_id":"` + f.variantID + `","quantity":"` + qty + `"}],` +
		`"payments":[{"method":"cash","amount":"` + cash + `"}]}`
}

// responseID reads the id of a response body.
func responseID(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct{ ID string }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body.ID
}
