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

func TestVoidPurchaseHandler(t *testing.T) {
	t.Run("pasa el motivo y responde la compra anulada con quién y por qué", func(t *testing.T) {
		p := aPurchase(t)
		require.NoError(t, p.Void(owner.UserID, "Factura duplicada", now))
		uc := &stubUseCases{purchase: p}

		rec := serve(t, uc, http.MethodPost, "/api/v1/purchases/"+p.ID().String()+"/void", `{"reason":"Factura duplicada"}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, "Factura duplicada", uc.reason)
		require.Equal(t, p.ID(), uc.id)
		var got map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Equal(t, "voided", got["status"])
		require.Equal(t, "0.00", got["balance_due"])
		require.Equal(t, "Factura duplicada", got["void_reason"])
		require.Equal(t, owner.UserID.String(), got["voided_by"])
	})

	t.Run("sin stock suficiente responde 409 con la línea", func(t *testing.T) {
		uc := &stubUseCases{err: &app.LineError{Index: 1, Err: domain.ErrInsufficientStock}}
		rec := serve(t, uc, http.MethodPost, "/api/v1/purchases/"+newID().String()+"/void", `{"reason":"Error"}`)
		body := requireError(t, rec, http.StatusConflict, "insufficient_stock")
		require.Equal(t, "1", body.Error.Details["line"])
		require.Contains(t, body.Error.Message, "Línea 2: ")
	})

	t.Run("mapea los demás errores de la anulación", func(t *testing.T) {
		for _, tc := range []struct {
			err    error
			status int
			code   string
		}{
			{domain.ErrPurchaseAlreadyVoided, http.StatusConflict, "purchase_already_voided"},
			{domain.ErrPurchaseHasPayments, http.StatusConflict, "purchase_has_payments"},
			{domain.ErrInvalidVoidReason, http.StatusUnprocessableEntity, "invalid_void_reason"},
			{domain.ErrPurchaseNotFound, http.StatusNotFound, "purchase_not_found"},
		} {
			rec := serve(t, &stubUseCases{err: tc.err}, http.MethodPost, "/api/v1/purchases/"+newID().String()+"/void", `{"reason":"Error"}`)
			requireError(t, rec, tc.status, tc.code)
		}
	})
}

func TestSupplierPaymentHandlers(t *testing.T) {
	t.Run("registra el pago con su fecha y responde lo que queda por pagar", func(t *testing.T) {
		p := aPurchase(t)
		pay, err := p.Pay(domain.PaymentInput{
			Method: domain.MethodNequi, Amount: dec("50000"), PaidOn: domain.NewDate(2026, time.October, 27), UserID: owner.UserID,
		}, now)
		require.NoError(t, err)
		uc := &stubUseCases{paid: app.SupplierPaymentResult{Payment: pay, Purchase: p}}

		rec := serve(t, uc, http.MethodPost, "/api/v1/purchases/"+p.ID().String()+"/payments",
			`{"method":"nequi","amount":"50000","paid_on":"2026-10-27"}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, "nequi", uc.payment.Method)
		require.Equal(t, "50000", uc.payment.Amount.String())
		require.Equal(t, domain.NewDate(2026, time.October, 27), *uc.payment.PaidOn)
		var got map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Equal(t, "50000.00", got["amount"])
		require.Equal(t, "57100.00", got["balance_after"])
		require.Equal(t, "2026-10-27", got["paid_on"])
	})

	t.Run("sin fecha la decide el caso de uso, y mapea sus errores", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrPaymentExceedsBalance}
		rec := serve(t, uc, http.MethodPost, "/api/v1/purchases/"+newID().String()+"/payments", `{"method":"cash","amount":"10"}`)
		requireError(t, rec, http.StatusConflict, "payment_exceeds_balance")
		require.Nil(t, uc.payment.PaidOn)

		for _, tc := range []struct {
			err    error
			status int
			code   string
		}{
			{domain.ErrPurchaseVoided, http.StatusConflict, "purchase_voided"},
			{domain.ErrInvalidPaymentAmount, http.StatusUnprocessableEntity, "invalid_payment_amount"},
			{domain.ErrPaymentDateInFuture, http.StatusUnprocessableEntity, "payment_date_in_future"},
			{domain.ErrPaymentBeforeInvoice, http.StatusUnprocessableEntity, "payment_before_invoice"},
		} {
			rec := serve(t, &stubUseCases{err: tc.err}, http.MethodPost, "/api/v1/purchases/"+newID().String()+"/payments", `{"method":"cash","amount":"10"}`)
			requireError(t, rec, tc.status, tc.code)
		}
	})

	t.Run("un monto que no es número es invalid_payment_amount", func(t *testing.T) {
		rec := serve(t, &stubUseCases{}, http.MethodPost, "/api/v1/purchases/"+newID().String()+"/payments", `{"method":"cash","amount":"diez"}`)
		require.Contains(t, []int{http.StatusBadRequest, http.StatusUnprocessableEntity}, rec.Code, rec.Body.String())
	})

	t.Run("lista lo que se debe a cada proveedor con los totales", func(t *testing.T) {
		supplier := newID()
		uc := &stubUseCases{payables: domain.PayablesPage{
			Items: []domain.SupplierPayable{{
				SupplierID: supplier, SupplierName: "Distribuidora del Valle", Balance: dec("214200"), Overdue: dec("107100"),
				DueSoon: dec("107100"), Purchases: 2, NextDue: domain.NewDate(2026, time.October, 25),
			}},
			Totals: domain.PayablesTotals{Balance: dec("214200"), Overdue: dec("107100"), DueSoon: dec("107100"), Suppliers: 1},
		}}

		rec := serve(t, uc, http.MethodGet, "/api/v1/payables?limit=10&offset=0", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, app.PayablesQuery{Limit: 10}, uc.payablesQuery)
		var got struct {
			Items  []map[string]any `json:"items"`
			Total  int              `json:"total"`
			Totals map[string]any   `json:"totals"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Equal(t, 1, got.Total)
		require.Equal(t, "107100.00", got.Items[0]["overdue"])
		require.Equal(t, "2026-10-25", got.Items[0]["next_due"])
		require.Equal(t, "214200.00", got.Totals["balance"])
	})
}
