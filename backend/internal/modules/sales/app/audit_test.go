package app_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestAuditOfSales(t *testing.T) {
	t.Run("anular una venta deja un evento sale.voided con su total", func(t *testing.T) {
		e := newEnv(t)
		location := newID()
		sale, err := e.sellFor(t, location, "52000", cash("52000"))
		require.NoError(t, err)

		_, err = e.svc.VoidSale(t.Context(), e.actor, sale.ID(), "Error del cajero")
		require.NoError(t, err)

		events := e.audit.of("sale.voided")
		require.Len(t, events, 1)
		require.Equal(t, sale.ID(), events[0].EntityID)
		require.Equal(t, location, events[0].LocationID)
		require.Equal(t, map[string]string{"total": "52000.00"}, events[0].Details)
	})

	t.Run("si el registro falla, la venta no se anula", func(t *testing.T) {
		e := newEnv(t)
		sale, err := e.sellFor(t, newID(), "52000", cash("52000"))
		require.NoError(t, err)
		e.audit.fail = errors.New("audit down")

		_, err = e.svc.VoidSale(t.Context(), e.actor, sale.ID(), "Error del cajero")
		require.ErrorContains(t, err, "audit down")
		stored, err := e.svc.GetSale(t.Context(), e.actor, sale.ID())
		require.NoError(t, err)
		require.NotEqual(t, domain.SaleVoided, stored.Status())
	})

	discounts := []struct {
		name, price, discount string
		want                  map[string]string
	}{
		{"ferretería: descuento de 2.500 en una caja de tornillos", "25000", "2500", map[string]string{"discount": "2500.00", "gross": "25000.00"}},
		{"almacén de ropa: descuento de 12.000 en una camisa", "80000", "12000", map[string]string{"discount": "12000.00", "gross": "80000.00"}},
		{"sin descuento no deja evento", "80000", "0", nil},
	}
	for _, tc := range discounts {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t)
			e.actor.UnlimitedDiscount = true
			id := e.addProduct("Producto", tc.price, "0", true)
			sale, err := e.svc.ConfirmSale(t.Context(), e.actor, app.ConfirmSaleInput{
				LocationID: newID(), Lines: []app.LineRequest{lineReq(id, "1", tc.discount)},
				Payments: []app.PaymentRequest{cash(tc.price)},
			})
			require.NoError(t, err)

			events := e.audit.of("sale.discount_applied")
			if tc.want == nil {
				require.Empty(t, events)
				return
			}
			require.Len(t, events, 1)
			require.Equal(t, sale.ID(), events[0].EntityID)
			require.Equal(t, tc.want, events[0].Details)
		})
	}
}

func TestAuditOfCashClosing(t *testing.T) {
	t.Run("cerrar con faltante deja un evento con el esperado, lo contado y la diferencia", func(t *testing.T) {
		e := newEnv(t)
		location := newID()
		session := e.openCash(t, location, "100000")

		_, err := e.svc.CloseCashSession(t.Context(), e.actor, session.ID(), app.CloseCashSessionInput{
			Counts: counts("50000", 1, "20000", 2),
		})
		require.NoError(t, err)

		events := e.audit.of("cash.closed_with_difference")
		require.Len(t, events, 1)
		require.Equal(t, session.ID(), events[0].EntityID)
		require.Equal(t, location, events[0].LocationID)
		require.Equal(t, map[string]string{"expected": "100000.00", "counted": "90000.00", "difference": "-10000.00"}, events[0].Details)
	})

	t.Run("cerrar cuadrada no deja evento", func(t *testing.T) {
		e := newEnv(t)
		session := e.openCash(t, newID(), "100000")

		_, err := e.svc.CloseCashSession(t.Context(), e.actor, session.ID(), app.CloseCashSessionInput{
			Counts: counts("100000", 1),
		})
		require.NoError(t, err)
		require.Empty(t, e.audit.entries)
	})

	t.Run("si el registro falla, la caja no se cierra", func(t *testing.T) {
		e := newEnv(t)
		session := e.openCash(t, newID(), "100000")
		e.audit.fail = errors.New("audit down")

		_, err := e.svc.CloseCashSession(t.Context(), e.actor, session.ID(), app.CloseCashSessionInput{
			Counts: counts("50000", 1),
		})
		require.ErrorContains(t, err, "audit down")
	})
}
