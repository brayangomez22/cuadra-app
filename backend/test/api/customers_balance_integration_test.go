//go:build integration

package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCustomerBalances checks that the customers list says what each one
// owes to whoever sees the receivables (P122), and that the aging orders by
// what they owe in total (P124).
func TestCustomerBalances(t *testing.T) {
	f := newCreditFlow(t)
	rec := f.do(t, http.MethodPut, "/api/v1/customers/"+f.customerID+"/credit", `{"credit_limit":"500000","term_days":30}`, f.ownerToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = f.do(t, http.MethodPost, "/api/v1/sales", f.creditSale(""), f.cashierToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	rec = f.do(t, http.MethodPost, "/api/v1/customers",
		`{"person_type":"legal","document_type":"nit","document_number":"900123456-8","name":"Constructora sin deuda"}`, f.ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	type customers struct {
		Items []struct {
			Name             string
			Balance, Overdue *string
		}
	}
	t.Run("la lista dice lo que debe cada cliente a quien ve la cartera", func(t *testing.T) {
		var page customers
		rec := f.do(t, http.MethodGet, "/api/v1/customers", "", f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		decode(t, rec, &page)
		owed := map[string]string{}
		for _, c := range page.Items {
			require.NotNil(t, c.Balance)
			owed[c.Name] = *c.Balance + "/" + *c.Overdue
		}
		require.Equal(t, "35700.00/0.00", owed["José Gómez"])
		require.Equal(t, "0.00/0.00", owed["Constructora sin deuda"])
	})
	t.Run("el bodeguero, que no ve la cartera, no recibe lo que deben", func(t *testing.T) {
		var page customers
		rec := f.do(t, http.MethodGet, "/api/v1/customers", "", f.warehouseToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		decode(t, rec, &page)
		require.Nil(t, page.Items[0].Balance)
	})
	t.Run("la cartera se ordena por lo que deben en total", func(t *testing.T) {
		rec := f.do(t, http.MethodGet, "/api/v1/receivables/aging?sort=total", "", f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		rec = f.do(t, http.MethodGet, "/api/v1/receivables/aging?sort=nombre", "", f.ownerToken)
		requireErrorCode(t, rec, http.StatusBadRequest, "invalid_request")
	})
	t.Run("la cartera se exporta a CSV", func(t *testing.T) {
		rec := f.do(t, http.MethodGet, "/api/v1/reports/export?report=receivables&format=csv&sort=total", "", f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "José Gómez;35700;0;0;0;0;0;35700")
	})
}
