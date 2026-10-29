package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func TestSuppliers(t *testing.T) {
	t.Run("no hay dos proveedores con el mismo NIT en el tenant", func(t *testing.T) {
		e := newEnv(t)
		e.addSupplier(t, "Distribuidora del Valle", "900123456-8", 30)
		_, err := e.svc.CreateSupplier(t.Context(), e.actor, supplierParams("Otra razón social", "900.123.456-8", 0))
		require.ErrorIs(t, err, domain.ErrNITTaken)
	})

	t.Run("el saldo por pagar del proveedor suma sus compras a crédito", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Confecciones Andinas", "901234567-7", 30)
		other := e.addSupplier(t, "Calzado del Norte", "900123456-8", 30)
		v := e.variant("Medias deportivas", "M / Blanco", "par", "0.19")
		for i, pay := range []domain.Payment{domain.PaymentCredit, domain.PaymentCredit, domain.PaymentCash} {
			_, err := e.svc.RegisterPurchase(t.Context(), e.actor,
				e.purchase(s.ID(), "FV-"+string(rune('1'+i)), pay, line(v, nil, "10", "1000")))
			require.NoError(t, err)
		}

		got, err := e.svc.GetSupplier(t.Context(), e.actor, s.ID())
		require.NoError(t, err)
		require.Equal(t, "23800.00", got.BalanceDue.StringFixed(2))

		page, err := e.svc.ListSuppliers(t.Context(), e.actor, app.SupplierQuery{})
		require.NoError(t, err)
		require.Equal(t, 2, page.Total)
		for _, item := range page.Items {
			if item.ID() == other.ID() {
				require.True(t, item.BalanceDue.IsZero())
			}
		}
	})

	t.Run("lista solo los activos salvo que pida otro estado", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Calzado del Norte", "900123456-8", 30)
		off := supplierParams("Calzado del Norte", "900123456-8", 30)
		off.Active = false
		_, err := e.svc.UpdateSupplier(t.Context(), e.actor, s.ID(), off)
		require.NoError(t, err)

		page, err := e.svc.ListSuppliers(t.Context(), e.actor, app.SupplierQuery{})
		require.NoError(t, err)
		require.Zero(t, page.Total)
		page, err = e.svc.ListSuppliers(t.Context(), e.actor, app.SupplierQuery{Status: domain.SuppliersAll})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total)
		_, err = e.svc.ListSuppliers(t.Context(), e.actor, app.SupplierQuery{Status: "x"})
		require.ErrorIs(t, err, app.ErrInvalidStatus)
	})

	t.Run("otro tenant no ve ni edita el proveedor", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Calzado del Norte", "900123456-8", 30)
		stranger := app.Actor{TenantID: newID(), UserID: newID()}
		_, err := e.svc.GetSupplier(t.Context(), stranger, s.ID())
		require.ErrorIs(t, err, domain.ErrSupplierNotFound)
		_, err = e.svc.UpdateSupplier(t.Context(), stranger, s.ID(), supplierParams("X", "900123456-8", 0))
		require.ErrorIs(t, err, domain.ErrSupplierNotFound)
		require.Equal(t, s.ID().String(), e.span(t, "purchases.UpdateSupplier")["supplier.id"].AsString())
	})
}

func TestPurchaseQueries(t *testing.T) {
	t.Run("lista las compras abiertas de un proveedor y lee una con sus líneas", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Distribuidora del Valle", "900123456-8", 30)
		v := e.variant("Tornillo drywall 6x1", "", "und", "0.19")
		credit, err := e.svc.RegisterPurchase(t.Context(), e.actor,
			e.purchase(s.ID(), "FE-1", domain.PaymentCredit, line(v, nil, "100", "450")))
		require.NoError(t, err)
		_, err = e.svc.RegisterPurchase(t.Context(), e.actor,
			e.purchase(s.ID(), "FE-2", domain.PaymentCash, line(v, nil, "100", "450")))
		require.NoError(t, err)

		sid := s.ID()
		page, err := e.svc.ListPurchases(t.Context(), e.actor, app.PurchaseQuery{SupplierID: &sid, Status: domain.PurchaseOpen})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total)
		require.Equal(t, credit.ID(), page.Items[0].ID)

		got, err := e.svc.GetPurchase(t.Context(), e.actor, credit.ID())
		require.NoError(t, err)
		require.Len(t, got.Lines(), 1)
		_, err = e.svc.ListPurchases(t.Context(), e.actor, app.PurchaseQuery{Status: "x"})
		require.ErrorIs(t, err, app.ErrInvalidStatus)
	})

	t.Run("otro tenant no ve la compra", func(t *testing.T) {
		e := newEnv(t)
		s := e.addSupplier(t, "Distribuidora del Valle", "900123456-8", 30)
		v := e.variant("Tornillo drywall 6x1", "", "und", "0.19")
		p, err := e.svc.RegisterPurchase(t.Context(), e.actor,
			e.purchase(s.ID(), "FE-1", domain.PaymentCash, line(v, nil, "1", "450")))
		require.NoError(t, err)
		_, err = e.svc.GetPurchase(t.Context(), app.Actor{TenantID: newID(), UserID: newID()}, p.ID())
		require.ErrorIs(t, err, domain.ErrPurchaseNotFound)
	})
}
