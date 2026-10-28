//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCustomers covers the customers over the whole API: the segments the
// sign-up seeds from the template, creating a customer as the POS does,
// finding it, and selling to it.
func TestCustomers(t *testing.T) {
	f := newSalesFlow(t)

	segments := func(t *testing.T) map[string]string {
		t.Helper()
		rec := f.do(t, http.MethodGet, "/api/v1/customer-segments", "", f.cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var list struct{ Items []struct{ ID, Name string } }
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		out := map[string]string{}
		for _, s := range list.Items {
			out[s.Name] = s.ID
		}
		return out
	}

	t.Run("el registro siembra los segmentos de la plantilla", func(t *testing.T) {
		got := segments(t)
		require.Len(t, got, 4)
		require.Contains(t, got, "Maestro de obra")
	})

	var customerID string
	t.Run("el cajero crea un cliente al vuelo y lo encuentra por documento y por nombre", func(t *testing.T) {
		body := `{"person_type":"natural","document_type":"cc","document_number":"1.020.304.050","name":"José Gómez",` +
			`"segment_id":"` + segments(t)["Maestro de obra"] + `"}`
		rec := f.do(t, http.MethodPost, "/api/v1/customers", body, f.cashierToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		customerID = responseID(t, rec)
		require.Contains(t, rec.Body.String(), `"document_number":"1020304050"`)

		rec = f.do(t, http.MethodPost, "/api/v1/customers", body, f.cashierToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "CC 1020304050")

		for _, q := range []string{"1020304050", "jose%20gomez"} {
			rec = f.do(t, http.MethodGet, "/api/v1/customers?q="+q, "", f.cashierToken)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.Contains(t, rec.Body.String(), customerID, q)
		}
	})

	t.Run("rechaza un NIT con dígito de verificación inválido", func(t *testing.T) {
		rec := f.do(t, http.MethodPost, "/api/v1/customers",
			`{"person_type":"legal","document_type":"nit","document_number":"900123456-1","name":"Empresa"}`, f.ownerToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "invalid_nit_check_digit")
	})

	t.Run("vende a un cliente activo y rechaza uno inactivo", func(t *testing.T) {
		rec := f.do(t, http.MethodPost, "/api/v1/cash-sessions",
			`{"location_id":"`+f.locationID+`","opening_float":"0"}`, f.cashierToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		withCustomer := strings.Replace(f.saleBody("1", "35700"), `{"location_id"`, `{"customer_id":"`+customerID+`","location_id"`, 1)

		rec = f.do(t, http.MethodPost, "/api/v1/sales", withCustomer, f.cashierToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"customer_id":"`+customerID+`"`)

		rec = f.do(t, http.MethodPut, "/api/v1/customers/"+customerID,
			`{"person_type":"natural","document_type":"cc","document_number":"1020304050","name":"José Gómez",`+
				`"segment_id":"`+segments(t)["Maestro de obra"]+`","active":false}`, f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		rec = f.do(t, http.MethodPost, "/api/v1/sales", withCustomer, f.cashierToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "customer_inactive")
	})

	t.Run("solo quien administra los clientes gestiona los segmentos", func(t *testing.T) {
		rec := f.do(t, http.MethodPost, "/api/v1/customer-segments", `{"name":"Mayorista"}`, f.cashierToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		rec = f.do(t, http.MethodPost, "/api/v1/customers",
			`{"person_type":"natural","document_type":"cc","document_number":"79845123","name":"Ana"}`, f.warehouseToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

		rec = f.do(t, http.MethodPost, "/api/v1/customer-segments", `{"name":"Mayorista"}`, f.ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		rec = f.do(t, http.MethodDelete, "/api/v1/customer-segments/"+segments(t)["Maestro de obra"], "", f.ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "customer_segment_in_use")
	})
}
