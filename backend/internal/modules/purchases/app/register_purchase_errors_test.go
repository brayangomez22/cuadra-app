package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func TestRegisterPurchaseRejections(t *testing.T) {
	t.Run("rechaza un proveedor inexistente o de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		v := e.variant("Tornillo drywall 6x1", "", "und", "0.19")
		_, err := e.svc.RegisterPurchase(t.Context(), e.actor,
			e.purchase(newID(), "FE-1", domain.PaymentCash, line(v, nil, "1", "450")))
		require.ErrorIs(t, err, domain.ErrSupplierNotFound)
	})

	t.Run("rechaza un proveedor inactivo", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Calzado del Norte", "900123456-8", 30)
		off := supplierParams("Calzado del Norte", "900123456-8", 30)
		off.Active = false
		_, err := e.svc.UpdateSupplier(t.Context(), e.actor, s.ID(), off)
		require.NoError(t, err)
		v := e.variant("Tenis de lona", "38 / Negro", "par", "0.19")
		_, err = e.svc.RegisterPurchase(t.Context(), e.actor,
			e.purchase(s.ID(), "FE-1", domain.PaymentCash, line(v, nil, "1", "45000")))
		require.ErrorIs(t, err, domain.ErrSupplierInactive)
	})

	t.Run("rechaza una variante inexistente o inactiva y dice en qué línea", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Distribuidora del Valle", "900123456-8", 30)
		ok := e.variant("Tornillo drywall 6x1", "", "und", "0.19")
		off := e.variant("Medias deportivas", "M / Blanco", "par", "0.19")
		info := e.catalog.variants[off]
		info.Active = false
		e.catalog.variants[off] = info
		for _, tc := range []struct {
			lines []app.PurchaseLineInput
			want  error
			index int
		}{
			{[]app.PurchaseLineInput{line(ok, nil, "1", "1"), line(newID(), nil, "1", "1")}, domain.ErrVariantNotFound, 1},
			{[]app.PurchaseLineInput{line(off, nil, "1", "1")}, domain.ErrVariantInactive, 0},
		} {
			_, err := e.svc.RegisterPurchase(t.Context(), e.actor, e.purchase(s.ID(), "FE-1", domain.PaymentCash, tc.lines...))
			require.ErrorIs(t, err, tc.want)
			var lineErr *app.LineError
			require.ErrorAs(t, err, &lineErr)
			require.Equal(t, tc.index, lineErr.Index)
		}
	})

	t.Run("rechaza unidad que la variante no tiene", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Distribuidora del Valle", "900123456-8", 30)
		v := e.variant("Tornillo drywall 6x1", "", "und", "0.19", unit("caja", "100"))
		dozen := unit("docena", "12")
		_, err := e.svc.RegisterPurchase(t.Context(), e.actor,
			e.purchase(s.ID(), "FE-1", domain.PaymentCash, line(v, &dozen, "1", "1")))
		require.ErrorIs(t, err, domain.ErrUnitNotAvailable)
	})

	t.Run("si una entrada falla no queda la compra", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Distribuidora del Valle", "900123456-8", 30)
		v := e.variant("Tornillo drywall 6x1", "", "und", "0.19")
		e.stock.err = domain.ErrLocationInactive
		_, err := e.svc.RegisterPurchase(t.Context(), e.actor,
			e.purchase(s.ID(), "FE-1", domain.PaymentCredit, line(v, nil, "1", "450")))
		require.ErrorIs(t, err, domain.ErrLocationInactive)
		require.Empty(t, e.purchases.rows)
		require.Zero(t, e.counter(t, "cuadra.purchases.registered", "", ""))
	})

	t.Run("las reglas de la factura las pone el dominio", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Distribuidora del Valle", "900123456-8", 30)
		v := e.variant("Tornillo drywall 6x1", "", "und", "0.19")
		in := e.purchase(s.ID(), "FE-1", domain.PaymentCash, line(v, nil, "1", "450"))
		in.IssuedOn = today.AddDays(1)
		_, err := e.svc.RegisterPurchase(t.Context(), e.actor, in)
		require.ErrorIs(t, err, domain.ErrInvoiceDateInFuture)
		var lineErr *app.LineError
		require.NotErrorAs(t, err, &lineErr)
	})
}
