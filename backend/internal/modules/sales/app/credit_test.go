package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// creditSale is a sale of one product of 10000 + 19% IVA (11900) to a
// customer, paying cashAmount in cash and the rest on credit.
func (e *env) creditSale(t *testing.T, cashAmount, creditAmount string) app.ConfirmSaleInput {
	t.Helper()
	id := e.addProduct("Producto", "10000", "0.19", true)
	customer := e.addCustomer(true)
	payments := []app.PaymentRequest{{Method: "credit", Amount: dec(creditAmount)}}
	if cashAmount != "0" {
		payments = append(payments, cash(cashAmount))
	}
	return app.ConfirmSaleInput{
		LocationID: newID(), CustomerID: &customer,
		Lines: []app.LineRequest{lineReq(id, "1", "0")}, Payments: payments,
	}
}

func TestConfirmSaleOnCredit(t *testing.T) {
	t.Run("la venta a crédito abre la cuenta por cobrar del cliente por la parte a crédito", func(t *testing.T) {
		e := newEnv(t)
		in := e.creditSale(t, "1900", "10000")
		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, in)
		require.NoError(t, err)
		require.Equal(t, []app.CreditCharge{{
			TenantID: e.actor.TenantID, UserID: e.actor.UserID, CustomerID: *in.CustomerID,
			SaleID: sale.ID(), Amount: dec("10000"),
		}}, e.credit.charges)
		require.Zero(t, e.authorizer.calls, "sin credenciales no se pide autorización")
	})

	t.Run("rechaza venta que excede el cupo sin autorización y no la guarda", func(t *testing.T) {
		e := newEnv(t)
		e.credit.chargeErr = &domain.CreditLimitExceededError{Available: dec("5000")}
		_, err := e.svc.ConfirmSale(t.Context(), e.actor, e.creditSale(t, "0", "11900"))
		require.ErrorIs(t, err, domain.ErrCreditLimitExceeded)
		require.Empty(t, e.store.sales)
	})

	t.Run("pasa a la cartera quién autorizó una venta sobre el cupo", func(t *testing.T) {
		e := newEnv(t)
		admin := newID()
		creds := app.Credentials{Email: "admin@negocio.co", Password: "clave-segura"}
		e.authorizer.users[creds] = admin
		in := e.creditSale(t, "0", "11900")
		in.CreditAuthorization = &creds
		_, err := e.svc.ConfirmSale(t.Context(), e.actor, in)
		require.NoError(t, err)
		require.Len(t, e.credit.charges, 1)
		require.Equal(t, &admin, e.credit.charges[0].AuthorizedBy)
	})

	t.Run("rechaza la autorización con credenciales inválidas sin guardar la venta", func(t *testing.T) {
		e := newEnv(t)
		in := e.creditSale(t, "0", "11900")
		in.CreditAuthorization = &app.Credentials{Email: "cajero@negocio.co", Password: "errada"}
		_, err := e.svc.ConfirmSale(t.Context(), e.actor, in)
		require.ErrorIs(t, err, domain.ErrCreditAuthorizationRejected)
		require.Empty(t, e.store.sales)
		require.Empty(t, e.credit.charges)
	})

	t.Run("una venta de contado no pasa por la cartera aunque traiga credenciales", func(t *testing.T) {
		e := newEnv(t)
		id := e.addProduct("Producto", "10000", "0.19", true)
		_, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(), Lines: []app.LineRequest{lineReq(id, "1", "0")},
			Payments:            []app.PaymentRequest{cash("11900")},
			CreditAuthorization: &app.Credentials{Email: "admin@negocio.co", Password: "x"},
		})
		require.NoError(t, err)
		require.Empty(t, e.credit.charges)
		require.Zero(t, e.authorizer.calls)
	})
}

func TestVoidSaleOnCredit(t *testing.T) {
	t.Run("anular una venta a crédito cancela su cuenta por cobrar", func(t *testing.T) {
		e := newEnv(t)
		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, e.creditSale(t, "0", "11900"))
		require.NoError(t, err)
		_, err = e.svc.VoidSale(t.Context(), e.actor, sale.ID(), "Error del cajero")
		require.NoError(t, err)
		require.Equal(t, []uuid.UUID{sale.ID()}, e.credit.canceled)
	})

	t.Run("rechaza anular una venta a crédito con abonos y la deja confirmada", func(t *testing.T) {
		e := newEnv(t)
		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, e.creditSale(t, "0", "11900"))
		require.NoError(t, err)
		e.credit.cancelErr = domain.ErrCreditSaleHasPayments
		_, err = e.svc.VoidSale(t.Context(), e.actor, sale.ID(), "Error del cajero")
		require.ErrorIs(t, err, domain.ErrCreditSaleHasPayments)
		stored, err := e.svc.GetSale(t.Context(), e.actor, sale.ID())
		require.NoError(t, err)
		require.Equal(t, domain.SaleConfirmed, stored.Status())
	})

	t.Run("anular una venta de contado no toca la cartera", func(t *testing.T) {
		e := newEnv(t)
		id := e.addProduct("Producto", "10000", "0.19", true)
		sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
			LocationID: newID(), Lines: []app.LineRequest{lineReq(id, "1", "0")},
			Payments: []app.PaymentRequest{cash("11900")},
		})
		require.NoError(t, err)
		_, err = e.svc.VoidSale(t.Context(), e.actor, sale.ID(), "Devolución")
		require.NoError(t, err)
		require.Empty(t, e.credit.canceled)
	})
}
