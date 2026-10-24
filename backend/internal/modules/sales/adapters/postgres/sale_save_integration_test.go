//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestSaleRepositorySave(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda y lee una venta confirmada con sus líneas, pagos y consecutivo", func(t *testing.T) {
		tn := f.createTenant(t)
		s := confirmed(t, tn, tn.location, now)

		f.save(t, s)
		require.Equal(t, int64(1), s.Number())

		got, err := f.sales.Get(t.Context(), tn.id, s.ID())
		require.NoError(t, err)
		require.Equal(t, normalized(s.Snapshot()), normalized(got.Snapshot()))
		require.Equal(t, int64(1), got.Number())
		require.Equal(t, s.CustomerID(), got.CustomerID())
		lines := got.Lines()
		require.Equal(t, tn.product, lines[0].VariantID())
		require.Empty(t, lines[0].VariantLabel())
		require.Equal(t, domain.VariantPriceRule, lines[0].PriceRule())
		require.Equal(t, tn.variant, lines[1].VariantID())
		require.Equal(t, tn.product, lines[1].ProductID())
		require.Equal(t, "25kg", lines[1].VariantLabel())
		require.Equal(t, domain.PriceRuleID("price_lists"), lines[1].PriceRule())
	})

	t.Run("rechaza una línea cuya variante es de otro producto", func(t *testing.T) {
		tn := f.createTenant(t)
		otherProduct := newID()
		s, err := domain.NewSale(domain.NewSaleParams{ID: newID(), TenantID: tn.id, LocationID: tn.location, UserID: newID(), CreatedAt: now})
		require.NoError(t, err)
		// The variant exists in the tenant, but belongs to another product.
		f.exec(t, tn.id, `INSERT INTO products (id, tenant_id, name, description, base_unit_id, tax_rate, active, created_at)
			VALUES ($1, $2, 'Arena fina', '', (SELECT id FROM units_of_measure WHERE code = 'und'), 0.19, true, $3)`,
			otherProduct, tn.id, now)
		require.NoError(t, s.AddLine(domain.LineInput{
			ProductID: otherProduct, VariantID: tn.variant, ProductName: "Arena fina", Quantity: dec("1"),
			UnitPrice: dec("1000"), TaxRate: dec("0"), Discount: dec("0"), PriceRule: domain.VariantPriceRule,
		}, domain.NoDiscountLimit))
		cash, err := domain.NewPayment(domain.PaymentCash, dec("1000"))
		require.NoError(t, err)
		require.NoError(t, s.Confirm([]domain.Payment{cash}, now))

		require.ErrorContains(t, f.sales.SaveConfirmed(t.Context(), s, noReserve), "sale_lines_variant_fkey")
		_, err = f.sales.Get(t.Context(), tn.id, s.ID())
		require.ErrorIs(t, err, domain.ErrSaleNotFound)
	})

	t.Run("si reserve falla, la venta no queda guardada y lo que escribió reserve se deshace", func(t *testing.T) {
		tn := f.createTenant(t)
		s := confirmed(t, tn, tn.location, now)
		boom := errors.New("sin stock")

		err := f.sales.SaveConfirmed(t.Context(), s, func(ctx context.Context) error {
			// reserve escribe en la misma transacción, como lo hace inventory.
			if err := f.db.WithTenantTx(ctx, tn.id, func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `INSERT INTO stock_levels (tenant_id, variant_id, location_id, quantity, average_cost, updated_at)
					VALUES ($1, $2, $3, -2.5, 0, $4)`, tn.id, tn.product, tn.location, now)
				return err
			}); err != nil {
				return err
			}
			return boom
		})
		require.ErrorIs(t, err, boom)
		require.Zero(t, s.Number())

		_, err = f.sales.Get(t.Context(), tn.id, s.ID())
		require.ErrorIs(t, err, domain.ErrSaleNotFound)
		var levels int
		require.NoError(t, f.db.WithTenantTx(t.Context(), tn.id, func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(), "SELECT count(*) FROM stock_levels").Scan(&levels)
		}))
		require.Zero(t, levels)
	})

	t.Run("una venta que hace rollback no deja hueco en el consecutivo", func(t *testing.T) {
		tn := f.createTenant(t)
		f.save(t, confirmed(t, tn, tn.location, now))
		require.Error(t, f.sales.SaveConfirmed(t.Context(), confirmed(t, tn, tn.location, now),
			func(context.Context) error { return errors.New("boom") }))

		third := confirmed(t, tn, tn.location, now)
		f.save(t, third)
		require.Equal(t, int64(2), third.Number())
	})

	t.Run("el consecutivo es independiente por sede", func(t *testing.T) {
		tn := f.createTenant(t)
		other := f.createLocation(t, tn.id)
		a1, b1, a2 := confirmed(t, tn, tn.location, now), confirmed(t, tn, other, now), confirmed(t, tn, tn.location, now)
		f.save(t, a1)
		f.save(t, b1)
		f.save(t, a2)
		require.Equal(t, []int64{1, 1, 2}, []int64{a1.Number(), b1.Number(), a2.Number()})
	})

	t.Run("dos ventas concurrentes en la misma sede no generan el mismo consecutivo", func(t *testing.T) {
		tn := f.createTenant(t)
		const n = 12
		sales := make([]*domain.Sale, n)
		for i := range sales {
			sales[i] = confirmed(t, tn, tn.location, now)
		}

		var wg sync.WaitGroup
		errs := make([]error, n)
		for i, s := range sales {
			wg.Go(func() { errs[i] = f.sales.SaveConfirmed(context.Background(), s, noReserve) })
		}
		wg.Wait()

		numbers := make([]int64, 0, n)
		for i, s := range sales {
			require.NoError(t, errs[i])
			numbers = append(numbers, s.Number())
		}
		slices.Sort(numbers)
		want := make([]int64, n)
		for i := range want {
			want[i] = int64(i + 1)
		}
		require.Equal(t, want, numbers, "consecutivos sin repetir ni saltar")
	})

	t.Run("rechaza una sede de otro tenant", func(t *testing.T) {
		tn, other := f.createTenant(t), f.createTenant(t)
		s := confirmed(t, tn, other.location, now)
		err := f.sales.SaveConfirmed(t.Context(), s, noReserve)
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})
}
