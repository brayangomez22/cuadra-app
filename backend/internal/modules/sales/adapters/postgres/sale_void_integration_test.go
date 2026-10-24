//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestSaleRepositoryVoid(t *testing.T) {
	f := newFixture(t)

	t.Run("anula una venta y corre change dentro de la transacción", func(t *testing.T) {
		tn := f.createTenant(t)
		s := confirmed(t, tn, tn.location, now)
		f.save(t, s)
		by := newID()

		voided, err := f.sales.Void(t.Context(), tn.id, s.ID(), func(_ context.Context, s *domain.Sale) error {
			return s.Void(by, "El cliente desistió", now.Add(time.Hour))
		})
		require.NoError(t, err)
		require.Equal(t, domain.SaleVoided, voided.Status())

		got, err := f.sales.Get(t.Context(), tn.id, s.ID())
		require.NoError(t, err)
		require.Equal(t, domain.SaleVoided, got.Status())
		require.Equal(t, &domain.VoidInfo{UserID: by, Reason: "El cliente desistió", At: now.Add(time.Hour)}, got.VoidInfo())
		require.Equal(t, int64(1), got.Number())
	})

	t.Run("si change falla, la venta sigue confirmada y lo que escribió se deshace", func(t *testing.T) {
		tn := f.createTenant(t)
		s := confirmed(t, tn, tn.location, now)
		f.save(t, s)
		boom := errors.New("boom")

		_, err := f.sales.Void(t.Context(), tn.id, s.ID(), func(ctx context.Context, s *domain.Sale) error {
			if err := s.Void(newID(), "Error", now); err != nil {
				return err
			}
			if err := f.db.WithTenantTx(ctx, tn.id, func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `INSERT INTO stock_levels (tenant_id, variant_id, location_id, quantity, average_cost, updated_at)
					VALUES ($1, $2, $3, 3.5, 0, $4)`, tn.id, tn.product, tn.location, now)
				return err
			}); err != nil {
				return err
			}
			return boom
		})
		require.ErrorIs(t, err, boom)

		got, err := f.sales.Get(t.Context(), tn.id, s.ID())
		require.NoError(t, err)
		require.Equal(t, domain.SaleConfirmed, got.Status())
		var levels int
		require.NoError(t, f.db.WithTenantTx(t.Context(), tn.id, func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(), "SELECT count(*) FROM stock_levels").Scan(&levels)
		}))
		require.Zero(t, levels)
	})

	t.Run("dos anulaciones concurrentes devuelven el stock una sola vez", func(t *testing.T) {
		tn := f.createTenant(t)
		s := confirmed(t, tn, tn.location, now)
		f.save(t, s)

		var reversals atomic.Int32
		var wg sync.WaitGroup
		errs := make([]error, 2)
		for i := range errs {
			wg.Go(func() {
				_, errs[i] = f.sales.Void(context.Background(), tn.id, s.ID(), func(_ context.Context, s *domain.Sale) error {
					if err := s.Void(newID(), "Duplicada", now); err != nil {
						return err
					}
					reversals.Add(1)
					// Mantiene el bloqueo un momento para que la otra anulación espere.
					time.Sleep(50 * time.Millisecond)
					return nil
				})
			})
		}
		wg.Wait()

		require.Equal(t, int32(1), reversals.Load())
		failed := 0
		for _, err := range errs {
			if err != nil {
				require.ErrorIs(t, err, domain.ErrSaleNotConfirmed)
				failed++
			}
		}
		require.Equal(t, 1, failed)
	})

	t.Run("anular una venta inexistente es sale_not_found", func(t *testing.T) {
		tn := f.createTenant(t)
		_, err := f.sales.Void(t.Context(), tn.id, newID(), func(context.Context, *domain.Sale) error { return nil })
		require.ErrorIs(t, err, domain.ErrSaleNotFound)
	})
}
