package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func TestRegisterPurchaseHandler(t *testing.T) {
	t.Run("traduce el body y responde la compra con montos en string", func(t *testing.T) {
		uc := &stubUseCases{purchase: aPurchase(t)}
		supplier, location, variant, unit := newID(), newID(), newID(), newID()
		body := `{"supplier_id":"` + supplier.String() + `","location_id":"` + location.String() + `",
			"invoice_number":"FE 1234","issued_on":"2026-10-20","payment":"credit","due_on":"2026-11-30",
			"lines":[{"variant_id":"` + variant.String() + `","unit_id":"` + unit.String() + `","quantity":"2",
			"unit_cost":"45000","tax_rate":"0.05"}]}`

		rec := serve(t, uc, http.MethodPost, "/api/v1/purchases", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		in := uc.input
		require.Equal(t, supplier, in.SupplierID)
		require.Equal(t, domain.NewDate(2026, time.October, 20), in.IssuedOn)
		require.Equal(t, domain.NewDate(2026, time.November, 30), *in.DueOn)
		require.Equal(t, domain.PaymentCredit, in.Payment)
		require.Equal(t, unit, *in.Lines[0].UnitID)
		require.Equal(t, "0.05", in.Lines[0].TaxRate.String())

		var got map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Equal(t, "open", got["status"])
		require.Equal(t, "107100.00", got["total"])
		require.Equal(t, "107100.00", got["balance_due"])
		require.Equal(t, "2026-11-19", got["due_on"])
		line := got["lines"].([]any)[0].(map[string]any)
		require.Equal(t, "caja", line["unit_code"])
		require.Equal(t, "100", line["unit_factor"])
		require.Equal(t, "2", line["quantity"])
		require.Equal(t, "45000.00", line["unit_cost"])
		require.Equal(t, "200", line["base_quantity"])
		require.Equal(t, "450.00", line["base_unit_cost"])
		taxes := got["taxes"].([]any)[0].(map[string]any)
		require.Equal(t, "0.19", taxes["rate"])
		require.Equal(t, "17100.00", taxes["tax"])
	})

	t.Run("un error de una línea dice cuál en details.line", func(t *testing.T) {
		uc := &stubUseCases{err: &app.LineError{Index: 2, Err: domain.ErrUnitNotAvailable}}
		rec := serve(t, uc, http.MethodPost, "/api/v1/purchases", validPurchaseBody())
		body := requireError(t, rec, http.StatusUnprocessableEntity, "unit_not_available")
		require.Equal(t, "2", body.Error.Details["line"])
		require.Contains(t, body.Error.Message, "Línea 3")
	})

	t.Run("mapea los errores de la compra", func(t *testing.T) {
		for _, tc := range []struct {
			err    error
			status int
			code   string
		}{
			{domain.ErrInvoiceTaken, http.StatusConflict, "purchase_invoice_taken"},
			{domain.ErrSupplierNotFound, http.StatusUnprocessableEntity, "supplier_not_found"},
			{domain.ErrSupplierInactive, http.StatusUnprocessableEntity, "supplier_inactive"},
			{domain.ErrLocationInactive, http.StatusUnprocessableEntity, "location_inactive"},
			{domain.ErrInvoiceDateInFuture, http.StatusUnprocessableEntity, "invoice_date_in_future"},
			{domain.ErrDueBeforeIssue, http.StatusUnprocessableEntity, "due_before_issue"},
			{domain.ErrTooManyLines, http.StatusUnprocessableEntity, "invalid_purchase_lines"},
		} {
			uc := &stubUseCases{err: tc.err}
			requireError(t, serve(t, uc, http.MethodPost, "/api/v1/purchases", validPurchaseBody()), tc.status, tc.code)
		}
	})

	t.Run("un costo que no es un monto es un costo inválido de su línea", func(t *testing.T) {
		uc := &stubUseCases{}
		body := `{"supplier_id":"` + newID().String() + `","location_id":"` + newID().String() + `",
			"invoice_number":"1","issued_on":"2026-10-20","payment":"cash",
			"lines":[{"variant_id":"` + newID().String() + `","quantity":"1","unit_cost":"veinte"}]}`
		got := requireError(t, serve(t, uc, http.MethodPost, "/api/v1/purchases", body), http.StatusUnprocessableEntity, "invalid_unit_cost")
		require.Equal(t, "0", got.Error.Details["line"])
	})
}

func TestPurchaseQueryHandlers(t *testing.T) {
	t.Run("lista con los filtros del query", func(t *testing.T) {
		supplier := newID()
		uc := &stubUseCases{purchases: app.PurchasePage{Items: []domain.PurchaseSummary{{
			ID: newID(), SupplierID: supplier, SupplierName: "Distribuidora del Valle", InvoiceNumber: "FE 1",
			IssuedOn: domain.NewDate(2026, time.October, 20), Payment: domain.PaymentCash, Total: dec("100"),
			BalanceDue: dec("0"), Lines: 3, CreatedAt: now,
		}}, Total: 1}}
		rec := serve(t, uc, http.MethodGet, "/api/v1/purchases?supplier_id="+supplier.String()+"&status=open&from=2026-10-01&to=2026-10-31", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		q := uc.purchaseQuery
		require.Equal(t, supplier, *q.SupplierID)
		require.Equal(t, domain.PurchaseOpen, q.Status)
		require.Equal(t, domain.NewDate(2026, time.October, 1), *q.From)
		require.Equal(t, domain.NewDate(2026, time.October, 31), *q.To)
		require.Equal(t, 20, q.Limit)
		require.Contains(t, rec.Body.String(), `"status":"paid"`)
		require.Contains(t, rec.Body.String(), `"lines":3`)
	})

	t.Run("status all lista todas", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, http.MethodGet, "/api/v1/purchases?status=all", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, domain.PurchaseStatus(""), uc.purchaseQuery.Status)
	})

	t.Run("una compra que no existe es 404", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrPurchaseNotFound}
		requireError(t, serve(t, uc, http.MethodGet, "/api/v1/purchases/"+newID().String(), ""), http.StatusNotFound, "purchase_not_found")
	})
}

func validPurchaseBody() string {
	return `{"supplier_id":"` + newID().String() + `","location_id":"` + newID().String() + `",
		"invoice_number":"FE 1","issued_on":"2026-10-20","payment":"cash",
		"lines":[{"variant_id":"` + newID().String() + `","quantity":"1","unit_cost":"100"}]}`
}
