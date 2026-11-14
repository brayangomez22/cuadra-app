//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestInvoicingDocumentData issues the document of a sale with the data the
// DIAN asks (T36a): the buyer's document type and municipality, and the SKU,
// unit and IVA treatment frozen in each line. The simulated provider refuses
// a document without them, as a real one does.
func TestInvoicingDocumentData(t *testing.T) {
	f := &salesFlow{}
	startJobs(t, f)
	do, owner := f.do, f.ownerToken
	disableCashSession(t, f)
	rec := do(t, http.MethodPut, "/api/v1/invoicing/settings", `{"enabled":true,"default_kind":"pos"}`, owner)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = do(t, http.MethodPost, "/api/v1/invoicing/resolutions", `{"kind":"pos","prefix":"pos","number":"18764000001234",`+
		`"range_from":1,"range_to":100,"valid_from":"2026-01-01","valid_to":"`+
		time.Now().AddDate(1, 0, 0).Format(time.DateOnly)+`"}`, owner)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var exemptVariant, customerID string

	t.Run("un producto exento y un cliente de otro país con municipio se guardan", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products", `{"sku":"cua-100","name":"Cuaderno 100 hojas","base_unit_id":"`+
			unitID(t, do, owner, "und")+`","cost":"3000","price":"4500","tax_rate":"0","tax_treatment":"exempt","attributes":{}}`, owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"tax_treatment":"exempt"`)
		var created struct {
			Variants []struct{ ID string } `json:"variants"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
		exemptVariant = created.Variants[0].ID
		rec = do(t, http.MethodPost, "/api/v1/inventory/receipts", `{"variant_id":"`+exemptVariant+`","location_id":"`+
			f.locationID+`","quantity":"5","unit_cost":"3000"}`, owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		rec = do(t, http.MethodPost, "/api/v1/customers", `{"person_type":"legal","document_type":"foreign_nit",`+
			`"document_number":"B12345678","name":"Textiles Ibéricos S.L.","municipality_code":"11001"}`, owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"municipality_code":"11001"`)
		customerID = responseID(t, rec)

		rec = do(t, http.MethodPost, "/api/v1/customers", `{"person_type":"natural","document_type":"cc",`+
			`"document_number":"1020304050","name":"Ana","municipality_code":"99999"}`, owner)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "invalid_municipality")
	})

	t.Run("la venta congela los datos de cada línea y el documento se emite", func(t *testing.T) {
		body := `{"location_id":"` + f.locationID + `","customer_id":"` + customerID + `","lines":[` +
			`{"variant_id":"` + f.variantID + `","quantity":"1"},{"variant_id":"` + exemptVariant + `","quantity":"2"}],` +
			`"payments":[{"method":"cash","amount":"44700"}]}`
		rec := do(t, http.MethodPost, "/api/v1/sales", body, owner)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var sale struct {
			ID    string
			Lines []struct {
				SKU          string `json:"sku"`
				BaseUnitCode string `json:"base_unit_code"`
				TaxTreatment string `json:"tax_treatment"`
			}
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sale))
		require.Equal(t, "CEM-50", sale.Lines[0].SKU)
		require.Equal(t, "bulto", sale.Lines[0].BaseUnitCode)
		require.Equal(t, "taxed", sale.Lines[0].TaxTreatment)
		require.Equal(t, "CUA-100", sale.Lines[1].SKU)
		require.Equal(t, "und", sale.Lines[1].BaseUnitCode)
		require.Equal(t, "exempt", sale.Lines[1].TaxTreatment)

		require.Eventually(t, func() bool {
			return documentsOf(t, f, sale.ID).Items[0].Status == "accepted"
		}, 15*time.Second, 100*time.Millisecond, "el proveedor simulado rechaza un documento sin sus datos")
	})

	t.Run("los municipios del DANE se listan", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/municipalities", "", f.cashierToken)
		require.Equal(t, http.StatusOK, rec.Code)
		var list struct{ Items []struct{ Code string } }
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list.Items, 1122)
	})
}
