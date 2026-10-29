package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func TestRegisterPurchase(t *testing.T) {
	t.Run("registrar una compra genera un movimiento de entrada por línea con el costo convertido a la unidad base", func(t *testing.T) {
		for _, tc := range []struct {
			business, product, label, base string
			unit                           app.UnitInfo
			qty, cost, baseQty, baseCost   string
		}{
			{"ferretería", "Tornillo drywall 6x1", "", "und", unit("caja", "100"), "2", "45000", "200", "450"},
			{"almacén de ropa", "Medias deportivas", "M / Blanco", "par", unit("docena", "12"), "3", "54000", "36", "4500"},
			{"droguería", "Acetaminofén 500 mg", "", "tableta", unit("caja", "30"), "1", "10000", "30", "333.3333"},
		} {
			e := newEnv(t)
			s := e.addSupplier(t, "Distribuidora "+tc.business, "900123456-8", 30)
			v := e.variant(tc.product, tc.label, tc.base, "0.19", tc.unit)
			p, err := e.svc.RegisterPurchase(t.Context(), e.actor,
				e.purchase(s.ID(), "FE-1", domain.PaymentCash, line(v, &tc.unit, tc.qty, tc.cost)))
			require.NoError(t, err, tc.business)

			require.Len(t, e.stock.calls, 1, tc.business)
			call := e.stock.calls[0]
			require.Equal(t, p.ID(), call.purchaseID)
			require.Equal(t, e.location, call.locationID)
			require.Len(t, call.lines, 1)
			require.True(t, call.lines[0].Factor.Equal(tc.unit.Factor), tc.business)

			l := p.Lines()[0]
			require.Equal(t, tc.unit.Code, l.Unit().Code, tc.business)
			require.Equal(t, tc.qty, l.Quantity().String(), tc.business)
			require.Equal(t, tc.cost, l.UnitCost().String(), tc.business)
			require.Equal(t, tc.baseQty, l.BaseQuantity().String(), tc.business)
			require.Equal(t, tc.baseCost, l.BaseUnitCost().String(), tc.business)
		}
	})

	t.Run("una línea en la unidad base entra con factor 1 y el nombre de la unidad base", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Papelería Mayorista", "900123456-8", 0)
		v := e.variant("Cuaderno cuadriculado 100 hojas", "", "und", "0.19")
		p, err := e.svc.RegisterPurchase(t.Context(), e.actor,
			e.purchase(s.ID(), "A-77", domain.PaymentCash, line(v, nil, "24", "3500")))
		require.NoError(t, err)
		l := p.Lines()[0]
		require.Nil(t, l.Unit().ID)
		require.Equal(t, "und", l.Unit().Code)
		require.Equal(t, "1", e.stock.calls[0].lines[0].Factor.String())
		require.Equal(t, "24", l.BaseQuantity().String())
	})

	t.Run("no se puede registrar dos veces el mismo número de factura del mismo proveedor", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Distribuidora del Valle", "900123456-8", 30)
		v := e.variant("Tornillo drywall 6x1", "", "und", "0.19")
		_, err := e.svc.RegisterPurchase(t.Context(), e.actor,
			e.purchase(s.ID(), "FE 1234", domain.PaymentCredit, line(v, nil, "100", "450")))
		require.NoError(t, err)

		_, err = e.svc.RegisterPurchase(t.Context(), e.actor,
			e.purchase(s.ID(), "fe-1234", domain.PaymentCredit, line(v, nil, "100", "450")))
		require.ErrorIs(t, err, domain.ErrInvoiceTaken)
		require.Len(t, e.stock.calls, 1, "la factura repetida no mueve el inventario")
		require.Equal(t, int64(1), e.counter(t, "cuadra.purchases.duplicate_invoices", "", ""))
	})

	t.Run("el mismo número de factura de otro proveedor sí se registra", func(t *testing.T) {
		e := newEnv(t)
		a := e.addSupplier(t, "Calzado del Norte", "900123456-8", 30)
		b := e.addSupplier(t, "Confecciones Andinas", "901234567-7", 15)
		v := e.variant("Medias deportivas", "M / Blanco", "par", "0.19")
		for _, s := range []*domain.Supplier{a, b} {
			_, err := e.svc.RegisterPurchase(t.Context(), e.actor,
				e.purchase(s.ID(), "FV-10", domain.PaymentCash, line(v, nil, "12", "4500")))
			require.NoError(t, err)
		}
	})

	t.Run("usa la tasa del producto si la línea no trae tasa, y la de la línea si la trae", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Droguería Mayorista", "900123456-8", 30)
		medicine := e.variant("Acetaminofén 500 mg", "", "tableta", "0")
		paint := e.variant("Vinilo blanco galón", "", "und", "0.19")
		withRate := line(paint, nil, "1", "100000")
		rate := dec("0.05")
		withRate.TaxRate = &rate
		p, err := e.svc.RegisterPurchase(t.Context(), e.actor, e.purchase(s.ID(), "D-1", domain.PaymentCash,
			line(medicine, nil, "100", "200"), withRate))
		require.NoError(t, err)
		require.Equal(t, "0", p.Lines()[0].TaxRate().String())
		require.Equal(t, "0.05", p.Lines()[1].TaxRate().String())
	})

	t.Run("cuenta la compra por medio de pago y sus líneas por unidad", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Distribuidora del Valle", "900123456-8", 30)
		box := unit("caja", "100")
		v := e.variant("Tornillo drywall 6x1", "", "und", "0.19", box)
		p, err := e.svc.RegisterPurchase(t.Context(), e.actor, e.purchase(s.ID(), "FE-9", domain.PaymentCredit,
			line(v, &box, "1", "45000"), line(v, nil, "50", "450")))
		require.NoError(t, err)
		require.Equal(t, int64(1), e.counter(t, "cuadra.purchases.registered", "payment", "credit"))
		require.Equal(t, int64(1), e.counter(t, "cuadra.purchases.lines", "unit", "alternate"))
		require.Equal(t, int64(1), e.counter(t, "cuadra.purchases.lines", "unit", "base"))
		attrs := e.span(t, "purchases.RegisterPurchase")
		require.Equal(t, p.ID().String(), attrs["purchase.id"].AsString())
		require.Equal(t, s.ID().String(), attrs["supplier.id"].AsString())
	})
}
