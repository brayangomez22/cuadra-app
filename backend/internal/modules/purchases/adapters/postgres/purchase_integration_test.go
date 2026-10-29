//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func TestPurchaseRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda y lee la compra con sus líneas en la unidad de compra y en la base", func(t *testing.T) {
		tn := f.createTenant(t)
		s := f.addSupplier(t, tn, "Distribuidora del Valle", "900123456-8")
		p := newPurchase(t, tn, s, "FE 1234", domain.PaymentCredit)
		f.save(t, p)

		got, err := f.purchases.GetByID(t.Context(), tn.id, p.ID())
		require.NoError(t, err)
		l := got.Lines()[0]
		require.Equal(t, tn.box, *l.Unit().ID)
		require.Equal(t, "caja", l.Unit().Code)
		require.True(t, l.Quantity().Equal(dec("2")))
		require.True(t, l.UnitCost().Equal(dec("45000")))
		require.True(t, l.BaseQuantity().Equal(dec("200")))
		require.True(t, l.BaseUnitCost().Equal(dec("450")))
		require.Equal(t, p.DueOn(), got.DueOn())
		require.True(t, got.Totals().Total.Equal(dec("107100")))
		require.True(t, got.BalanceDue().Equal(dec("107100")))
		require.Equal(t, "Llegó completa", got.Notes())
	})

	t.Run("la factura repetida del mismo proveedor se rechaza antes de recibir el inventario", func(t *testing.T) {
		tn := f.createTenant(t)
		s := f.addSupplier(t, tn, "Distribuidora del Valle", "900123456-8")
		f.save(t, newPurchase(t, tn, s, "FE 1234", domain.PaymentCash))

		dup := newPurchase(t, tn, s, "fe-1234", domain.PaymentCash)
		received := false
		err := f.purchases.Save(t.Context(), dup, func(context.Context) error { received = true; return nil })
		require.ErrorIs(t, err, domain.ErrInvoiceTaken)
		require.False(t, received)

		other := f.addSupplier(t, tn, "Confecciones Andinas", "901234567-7")
		f.save(t, newPurchase(t, tn, other, "FE 1234", domain.PaymentCash))
	})

	t.Run("la entrada fallida deshace la compra", func(t *testing.T) {
		tn := f.createTenant(t)
		s := f.addSupplier(t, tn, "Distribuidora del Valle", "900123456-8")
		p := newPurchase(t, tn, s, "FE-9", domain.PaymentCash)
		boom := errors.New("sede inactiva")
		require.ErrorIs(t, f.purchases.Save(t.Context(), p, func(context.Context) error { return boom }), boom)
		_, err := f.purchases.GetByID(t.Context(), tn.id, p.ID())
		require.ErrorIs(t, err, domain.ErrPurchaseNotFound)
	})

	t.Run("lista por proveedor, estado y fechas, y suma el saldo por pagar de cada proveedor", func(t *testing.T) {
		tn := f.createTenant(t)
		a := f.addSupplier(t, tn, "Distribuidora del Valle", "900123456-8")
		b := f.addSupplier(t, tn, "Confecciones Andinas", "901234567-7")
		f.save(t, newPurchase(t, tn, a, "A-1", domain.PaymentCredit))
		f.save(t, newPurchase(t, tn, a, "A-2", domain.PaymentCredit))
		f.save(t, newPurchase(t, tn, b, "B-1", domain.PaymentCash))

		aid := a.ID()
		items, total, err := f.purchases.Search(t.Context(), domain.PurchaseFilter{
			TenantID: tn.id, SupplierID: &aid, Status: domain.PurchaseOpen, Limit: 20,
		})
		require.NoError(t, err)
		require.Equal(t, 2, total)
		require.Equal(t, "Distribuidora del Valle", items[0].SupplierName)
		require.Equal(t, 1, items[0].Lines)
		require.NotNil(t, items[0].DueOn)

		day := domain.NewDate(2026, time.October, 21)
		_, total, err = f.purchases.Search(t.Context(), domain.PurchaseFilter{TenantID: tn.id, From: &day, Limit: 20})
		require.NoError(t, err)
		require.Zero(t, total)
		_, total, err = f.purchases.Search(t.Context(), domain.PurchaseFilter{TenantID: tn.id, To: &day, Status: domain.PurchasePaid, Limit: 20})
		require.NoError(t, err)
		require.Equal(t, 1, total)

		balances, err := f.purchases.BalancesDue(t.Context(), tn.id, []uuid.UUID{a.ID(), b.ID()})
		require.NoError(t, err)
		require.True(t, balances[a.ID()].Equal(dec("214200")), balances[a.ID()].String())
		_, owed := balances[b.ID()]
		require.False(t, owed)
	})

	t.Run("aislamiento: el tenant B no ve las compras ni los saldos del A ni registra a su proveedor", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		s := f.addSupplier(t, a, "Distribuidora del Valle", "900123456-8")
		p := newPurchase(t, a, s, "FE-1", domain.PaymentCredit)
		f.save(t, p)

		_, err := f.purchases.GetByID(t.Context(), b.id, p.ID())
		require.ErrorIs(t, err, domain.ErrPurchaseNotFound)
		_, total, err := f.purchases.Search(t.Context(), domain.PurchaseFilter{TenantID: b.id, Limit: 20})
		require.NoError(t, err)
		require.Zero(t, total)
		balances, err := f.purchases.BalancesDue(t.Context(), b.id, []uuid.UUID{s.ID()})
		require.NoError(t, err)
		require.Empty(t, balances)

		// Una compra del tenant B con el proveedor del A no pasa las llaves foráneas.
		snap := s.Snapshot()
		snap.TenantID = b.id
		forged, err := domain.RehydrateSupplier(snap)
		require.NoError(t, err)
		stolen := newPurchase(t, b, forged, "FE-2", domain.PaymentCash)
		require.ErrorIs(t, f.purchases.Save(t.Context(), stolen, receive(stolen)), domain.ErrSupplierNotFound)
	})
}
