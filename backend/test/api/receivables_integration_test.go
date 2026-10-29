//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// creditFlow is a salesFlow with a customer and the cashier's open cash
// session.
type creditFlow struct {
	*salesFlow
	customerID string
}

func newCreditFlow(t *testing.T) *creditFlow {
	t.Helper()
	f := &creditFlow{salesFlow: newSalesFlow(t)}
	rec := f.do(t, http.MethodPost, "/api/v1/customers",
		`{"person_type":"natural","document_type":"cc","document_number":"1020304050","name":"José Gómez"}`, f.cashierToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	f.customerID = responseID(t, rec)
	rec = f.do(t, http.MethodPost, "/api/v1/cash-sessions", `{"location_id":"`+f.locationID+`","opening_float":"0"}`, f.cashierToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	return f
}

// creditSale is a sale of one unit (35700 with IVA) to the customer, all on
// credit, with the authorization extra (`,"credit_authorization":{...}`).
func (f *creditFlow) creditSale(extra string) string {
	return `{"location_id":"` + f.locationID + `","customer_id":"` + f.customerID + `",` +
		`"lines":[{"variant_id":"` + f.variantID + `","quantity":"1"}],` +
		`"payments":[{"method":"credit","amount":"35700"}]` + extra + `}`
}

// credit returns the customer's balance and available credit.
func (f *creditFlow) credit(t *testing.T) (balance, available string) {
	t.Helper()
	rec := f.do(t, http.MethodGet, "/api/v1/customers/"+f.customerID+"/credit", "", f.cashierToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct{ Balance, Available string }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body.Balance, body.Available
}

// TestReceivables covers credit over the whole API: the limit, credit sales
// within and over it, payments (cash into the drawer), the statement, the
// aging and voiding a credit sale.
func TestReceivables(t *testing.T) {
	f := newCreditFlow(t)

	t.Run("un cliente sin cupo no puede comprar a crédito", func(t *testing.T) {
		_, available := f.credit(t)
		require.Equal(t, "0.00", available)
		rec := f.do(t, http.MethodPost, "/api/v1/sales", f.creditSale(""), f.cashierToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "credit_not_enabled")
	})

	t.Run("solo el administrador asigna el cupo", func(t *testing.T) {
		body := `{"credit_limit":"50000","term_days":30}`
		rec := f.do(t, http.MethodPut, "/api/v1/customers/"+f.customerID+"/credit", body, f.cashierToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		rec = f.do(t, http.MethodPut, "/api/v1/customers/"+f.customerID+"/credit", body, f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"available":"50000.00"`)
	})

	var firstSale string
	t.Run("una venta a crédito dentro del cupo abre la cuenta por cobrar", func(t *testing.T) {
		rec := f.do(t, http.MethodPost, "/api/v1/sales", f.creditSale(""), f.cashierToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		firstSale = responseID(t, rec)
		require.Contains(t, rec.Body.String(), `"method":"credit"`)
		balance, available := f.credit(t)
		require.Equal(t, "35700.00", balance)
		require.Equal(t, "14300.00", available)
	})

	t.Run("una venta que excede el cupo se rechaza sin autorización y dice el disponible", func(t *testing.T) {
		rec := f.do(t, http.MethodPost, "/api/v1/sales", f.creditSale(""), f.cashierToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "credit_limit_exceeded")
		require.Contains(t, rec.Body.String(), `"available":"14300.00"`)

		wrong := `,"credit_authorization":{"email":"ana@ventas.co","password":"no-es-la-clave"}`
		rec = f.do(t, http.MethodPost, "/api/v1/sales", f.creditSale(wrong), f.cashierToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "credit_authorization_rejected")
		require.Equal(t, "9", f.stockOf(t, f.variantID), "la venta rechazada no sacó stock")
	})

	var secondSale string
	t.Run("la dueña autoriza una venta sobre el cupo y la cuenta guarda quién", func(t *testing.T) {
		ok := `,"credit_authorization":{"email":"ana@ventas.co","password":"clave-segura-123"}`
		rec := f.do(t, http.MethodPost, "/api/v1/sales", f.creditSale(ok), f.cashierToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		secondSale = responseID(t, rec)
		balance, available := f.credit(t)
		require.Equal(t, "71400.00", balance)
		require.Equal(t, "0.00", available)
		rec = f.do(t, http.MethodGet, "/api/v1/customers/"+f.customerID+"/credit", "", f.cashierToken)
		require.Contains(t, rec.Body.String(), `"authorized_by"`)
	})

	t.Run("un abono en efectivo reduce el saldo y entra a la caja del cajero", func(t *testing.T) {
		rec := f.do(t, http.MethodPost, "/api/v1/customers/"+f.customerID+"/payments",
			`{"method":"cash","amount":"20000"}`, f.cashierToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"balance":"51400.00"`)
		require.Contains(t, rec.Body.String(), `"cash_session_id"`)

		rec = f.do(t, http.MethodGet, "/api/v1/cash-sessions/current", "", f.cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "Abono de cartera")

		rec = f.do(t, http.MethodPost, "/api/v1/customers/"+f.customerID+"/payments",
			`{"method":"transfer","amount":"51400.01"}`, f.cashierToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "payment_exceeds_balance")
	})

	t.Run("el estado de cuenta y la cartera por edades muestran lo que debe", func(t *testing.T) {
		rec := f.do(t, http.MethodGet, "/api/v1/customers/"+f.customerID+"/credit/movements", "", f.cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var page struct {
			Total int
			Items []struct{ Kind, Balance string }
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		require.Equal(t, 3, page.Total)
		require.Equal(t, "payment", page.Items[0].Kind)
		require.Equal(t, "51400.00", page.Items[0].Balance)

		rec = f.do(t, http.MethodGet, "/api/v1/receivables/aging", "", f.cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"name":"José Gómez"`)
		require.Contains(t, rec.Body.String(), `"current":"51400.00"`)

		rec = f.do(t, http.MethodGet, "/api/v1/receivables/aging", "", f.warehouseToken)
		require.Equal(t, http.StatusForbidden, rec.Code, "el bodeguero no ve la cartera")
	})

	t.Run("anular una venta a crédito abonada se rechaza y una sin abonos libera su saldo", func(t *testing.T) {
		rec := f.do(t, http.MethodPost, "/api/v1/sales/"+firstSale+"/void", `{"reason":"Error del cajero"}`, f.ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "credit_sale_has_payments")

		rec = f.do(t, http.MethodPost, "/api/v1/sales/"+secondSale+"/void", `{"reason":"Error del cajero"}`, f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		balance, _ := f.credit(t)
		require.Equal(t, "15700.00", balance)
	})
}
