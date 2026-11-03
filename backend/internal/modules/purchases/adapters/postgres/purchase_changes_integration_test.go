//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func payment(amount string) domain.PaymentInput {
	return domain.PaymentInput{
		Method: domain.MethodTransfer, Amount: dec(amount), PaidOn: domain.NewDate(2026, time.October, 27), UserID: newID(),
	}
}

func TestPurchaseRepositoryVoid(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda la compra anulada con quién, cuándo y por qué", func(t *testing.T) {
		tn := f.createTenant(t)
		p := newPurchase(t, tn, f.addSupplier(t, tn, "Distribuidora del Valle", "900123456-8"), "FE-1", domain.PaymentCredit)
		f.save(t, p)
		by := newID()

		voided, err := f.purchases.Void(t.Context(), tn.id, p.ID(), func(_ context.Context, p *domain.Purchase) error {
			return p.Void(by, "Factura duplicada", now)
		})
		require.NoError(t, err)
		require.Equal(t, domain.PurchaseVoided, voided.Status())

		got, err := f.purchases.GetByID(t.Context(), tn.id, p.ID())
		require.NoError(t, err)
		require.Equal(t, &domain.Voiding{By: by, Reason: "Factura duplicada", At: now}, got.Voided())
		require.True(t, got.BalanceDue().IsZero())

		page, _, err := f.purchases.Search(t.Context(), domain.PurchaseFilter{TenantID: tn.id, Status: domain.PurchaseVoided, Limit: 10})
		require.NoError(t, err)
		require.Len(t, page, 1)
		require.Equal(t, domain.PurchaseVoided, page[0].Status())
		paid, _, err := f.purchases.Search(t.Context(), domain.PurchaseFilter{TenantID: tn.id, Status: domain.PurchasePaid, Limit: 10})
		require.NoError(t, err)
		require.Empty(t, paid)
	})

	t.Run("si falla lo que hace dentro de la transacción, la compra sigue igual", func(t *testing.T) {
		tn := f.createTenant(t)
		p := newPurchase(t, tn, f.addSupplier(t, tn, "Distribuidora del Valle", "900123456-8"), "FE-1", domain.PaymentCash)
		f.save(t, p)
		boom := errors.New("ya se vendió")

		_, err := f.purchases.Void(t.Context(), tn.id, p.ID(), func(_ context.Context, p *domain.Purchase) error {
			require.NoError(t, p.Void(newID(), "Error", now))
			return boom
		})
		require.ErrorIs(t, err, boom)

		got, err := f.purchases.GetByID(t.Context(), tn.id, p.ID())
		require.NoError(t, err)
		require.Equal(t, domain.PurchasePaid, got.Status())
	})

	t.Run("otra empresa no anula ni paga la compra", func(t *testing.T) {
		tn, other := f.createTenant(t), f.createTenant(t)
		p := newPurchase(t, tn, f.addSupplier(t, tn, "Distribuidora del Valle", "900123456-8"), "FE-1", domain.PaymentCredit)
		f.save(t, p)

		_, err := f.purchases.Void(t.Context(), other.id, p.ID(), func(_ context.Context, p *domain.Purchase) error {
			return p.Void(newID(), "Error", now)
		})
		require.ErrorIs(t, err, domain.ErrPurchaseNotFound)
		_, _, err = f.purchases.Pay(t.Context(), other.id, p.ID(), func(p *domain.Purchase) (*domain.SupplierPayment, error) {
			return p.Pay(payment("1000"), now)
		})
		require.ErrorIs(t, err, domain.ErrPurchaseNotFound)
		payments, err := f.purchases.Payments(t.Context(), other.id, p.ID())
		require.NoError(t, err)
		require.Empty(t, payments)
	})
}

func TestPurchaseRepositoryPayments(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda el pago y el saldo nuevo juntos y los lee en orden", func(t *testing.T) {
		tn := f.createTenant(t)
		p := newPurchase(t, tn, f.addSupplier(t, tn, "Distribuidora del Valle", "900123456-8"), "FE-1", domain.PaymentCredit)
		f.save(t, p)

		var payments []*domain.SupplierPayment
		for _, amount := range []string{"50000", "7100"} {
			_, pay, err := f.purchases.Pay(t.Context(), tn.id, p.ID(), func(p *domain.Purchase) (*domain.SupplierPayment, error) {
				return p.Pay(payment(amount), now)
			})
			require.NoError(t, err)
			payments = append(payments, pay)
		}

		got, err := f.purchases.GetByID(t.Context(), tn.id, p.ID())
		require.NoError(t, err)
		require.True(t, got.BalanceDue().Equal(dec("50000")), got.BalanceDue().String())
		stored, err := f.purchases.Payments(t.Context(), tn.id, p.ID())
		require.NoError(t, err)
		require.Len(t, stored, 2)
		for i := range payments {
			require.Equal(t, payments[i].Snapshot().ID, stored[i].ID())
			require.True(t, payments[i].Amount().Equal(stored[i].Amount()))
			require.True(t, payments[i].BalanceAfter().Equal(stored[i].BalanceAfter()))
			require.Equal(t, payments[i].PaidOn(), stored[i].PaidOn())
		}
	})
}

func TestPurchaseRepositoryPayables(t *testing.T) {
	f := newFixture(t)

	t.Run("suma lo que se debe a cada proveedor por vencimiento y deja fuera lo pagado y lo anulado", func(t *testing.T) {
		tn := f.createTenant(t)
		valle := f.addSupplier(t, tn, "Distribuidora del Valle", "900123456-8")
		andes := f.addSupplier(t, tn, "Confecciones Andinas", "901234567-7")
		// Todas vencen el 19 de noviembre: 30 días después del 20 de octubre.
		f.save(t, newPurchase(t, tn, valle, "FE-1", domain.PaymentCredit))
		f.save(t, newPurchase(t, tn, valle, "FE-2", domain.PaymentCredit))
		f.save(t, newPurchase(t, tn, andes, "C-1", domain.PaymentCredit))
		f.save(t, newPurchase(t, tn, andes, "C-2", domain.PaymentCash))
		voided := newPurchase(t, tn, andes, "C-3", domain.PaymentCredit)
		f.save(t, voided)
		_, err := f.purchases.Void(t.Context(), tn.id, voided.ID(), func(_ context.Context, p *domain.Purchase) error {
			return p.Void(newID(), "Error", now)
		})
		require.NoError(t, err)
		_, _, err = f.purchases.Pay(t.Context(), tn.id, voided.ID(), func(p *domain.Purchase) (*domain.SupplierPayment, error) {
			return p.Pay(payment("1"), now)
		})
		require.ErrorIs(t, err, domain.ErrPurchaseVoided)

		soon, err := f.purchases.Payables(t.Context(), tn.id, domain.PayablesWindowOn(domain.NewDate(2026, time.November, 15)), 10, 0)
		require.NoError(t, err)
		require.Len(t, soon.Items, 2)
		require.Equal(t, valle.ID(), soon.Items[0].SupplierID) // debe más
		require.Equal(t, "Distribuidora del Valle", soon.Items[0].SupplierName)
		require.True(t, soon.Items[0].Balance.Equal(dec("214200")))
		require.True(t, soon.Items[0].DueSoon.Equal(dec("214200")))
		require.True(t, soon.Items[0].Overdue.IsZero())
		require.Equal(t, 2, soon.Items[0].Purchases)
		require.Equal(t, domain.NewDate(2026, time.November, 19), soon.Items[0].NextDue)
		require.True(t, soon.Totals.Balance.Equal(dec("321300")))
		require.Equal(t, 2, soon.Totals.Suppliers)

		late, err := f.purchases.Payables(t.Context(), tn.id, domain.PayablesWindowOn(domain.NewDate(2026, time.November, 25)), 10, 0)
		require.NoError(t, err)
		require.True(t, late.Totals.Overdue.Equal(dec("321300")))
		require.True(t, late.Totals.DueSoon.IsZero())

		other, err := f.purchases.Payables(t.Context(), f.createTenant(t).id, domain.PayablesWindowOn(domain.NewDate(2026, time.November, 15)), 10, 0)
		require.NoError(t, err)
		require.Empty(t, other.Items)
		require.True(t, other.Totals.Balance.IsZero())
	})
}
