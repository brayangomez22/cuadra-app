//go:build integration

package postgres_test

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestKardex(t *testing.T) {
	f := newFixture(t)

	t.Run("el kardex reconstruye el stock actual", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		bodega := f.createLocation(t, tenant, "Bodega")
		centro := f.createLocation(t, tenant, "Centro")
		user := newID()

		f.receive(t, tenant, variant, bodega.ID(), "10", "1000")
		f.receive(t, tenant, variant, bodega.ID(), "2.5", "1200")
		require.NoError(t, f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{bodega.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Adjust(dec("-1.25"), "Bulto roto", domain.StockPolicy{}, info(user))
				return []*domain.StockMovement{m}, err
			}))
		require.NoError(t, f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{bodega.ID(), centro.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				out, in, err := domain.Transfer(levels[0], levels[1], dec("3"), domain.StockPolicy{}, user, now)
				return []*domain.StockMovement{out, in}, err
			}))

		for _, loc := range []*domain.Location{bodega, centro} {
			locID := loc.ID()
			kardex, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{VariantID: variant, LocationID: &locID, Limit: 100})
			require.NoError(t, err)
			sum := decimal.Zero
			for _, m := range kardex.Items {
				require.Equal(t, locID, m.LocationID())
				sum = sum.Add(m.Quantity())
			}
			current := f.level(t, tenant, variant, locID)
			require.True(t, sum.Equal(current), "%s: kardex %s, level %s", loc.Name(), sum, current)
			require.True(t, kardex.Items[0].BalanceAfter().Equal(current), "the newest movement carries the balance")
		}
		require.True(t, dec("8.25").Equal(f.level(t, tenant, variant, bodega.ID())))
		require.True(t, dec("3").Equal(f.level(t, tenant, variant, centro.ID())))
	})

	t.Run("devuelve los movimientos completos, el más reciente primero y paginados", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		bodega := f.createLocation(t, tenant, "Bodega")
		centro := f.createLocation(t, tenant, "Centro")
		user := newID()
		f.receive(t, tenant, variant, bodega.ID(), "10", "1000")
		require.NoError(t, f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{bodega.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Adjust(dec("-1"), "Conteo físico", domain.StockPolicy{}, info(user))
				return []*domain.StockMovement{m}, err
			}))
		var transferOut *domain.StockMovement
		require.NoError(t, f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{bodega.ID(), centro.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				out, in, err := domain.Transfer(levels[0], levels[1], dec("2"), domain.StockPolicy{}, user, now)
				transferOut = out
				return []*domain.StockMovement{out, in}, err
			}))

		page, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{VariantID: variant, Limit: 2, Offset: 1})
		require.NoError(t, err)
		require.Equal(t, 4, page.Total)
		require.Len(t, page.Items, 2)
		got := page.Items[0]
		require.Equal(t, transferOut.ID(), got.ID())
		require.Equal(t, domain.MovementTransferOut, got.Type())
		require.Equal(t, transferOut.Reference(), got.Reference())
		require.True(t, dec("-2").Equal(got.Quantity()))
		require.True(t, dec("1000").Equal(got.UnitCost()))
		require.True(t, dec("7").Equal(got.BalanceAfter()))
		require.True(t, dec("1000").Equal(got.AverageCostAfter()))
		require.Equal(t, user, got.UserID())
		require.Equal(t, now, got.OccurredAt())
		adjustment := page.Items[1]
		require.Equal(t, domain.MovementAdjustment, adjustment.Type())
		require.Equal(t, "Conteo físico", adjustment.Reason())
		require.Nil(t, adjustment.Reference())
	})
}

func TestStockLevelsByVariantAndLot(t *testing.T) {
	f := newFixture(t)

	t.Run("dos variantes del mismo producto tienen niveles y kardex separados", func(t *testing.T) {
		// Ropa: talla M y talla L de la misma camiseta.
		tenant := f.createTenant(t)
		variants := f.createProduct(t, tenant, 1)
		sizeM, sizeL := variants[0], variants[1]
		loc := f.createLocation(t, tenant, "Almacén")

		f.receive(t, tenant, sizeM, loc.ID(), "6", "15000")
		f.receive(t, tenant, sizeL, loc.ID(), "4", "16000")

		require.True(t, dec("6").Equal(f.level(t, tenant, sizeM, loc.ID())))
		require.True(t, dec("4").Equal(f.level(t, tenant, sizeL, loc.ID())))
		kardex, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{VariantID: sizeL, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 1, kardex.Total)
		require.Equal(t, sizeL, kardex.Items[0].VariantID())
	})

	t.Run("dos EnsureStockLevel concurrentes con lot_id NULL no crean dos niveles", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")

		const workers = 8
		start := make(chan struct{})
		errs := make([]error, workers)
		var wg sync.WaitGroup
		for i := range workers {
			wg.Go(func() {
				<-start
				errs[i] = f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{loc.ID()},
					func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
						m, err := levels[0].Receive(dec("1"), dec("100"), info(newID()))
						return []*domain.StockMovement{m}, err
					})
			})
		}
		close(start)
		wg.Wait()

		for _, err := range errs {
			require.NoError(t, err)
		}
		require.Equal(t, 1, f.levelRows(t, tenant, variant, loc.ID()))
		require.True(t, dec("8").Equal(f.level(t, tenant, variant, loc.ID())))
	})

	t.Run("la base rechaza un segundo nivel con la misma variante, sede y lote NULL", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")
		f.receive(t, tenant, variant, loc.ID(), "1", "100")

		err := f.db.WithTenantTx(t.Context(), tenant, func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), `INSERT INTO stock_levels (tenant_id, variant_id, location_id, lot_id, quantity, average_cost, updated_at)
				VALUES ($1, $2, $3, NULL, 0, 0, $4)`, tenant, variant, loc.ID(), now)
			return err
		})
		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		require.Equal(t, "stock_levels_key", pgErr.ConstraintName)
	})

	t.Run("mover y listar el stock sin lote no toca ni muestra el nivel de un lote", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")
		lot := newID()
		f.exec(t, tenant, `INSERT INTO stock_levels (tenant_id, variant_id, location_id, lot_id, quantity, average_cost, updated_at)
			VALUES ($1, $2, $3, $4, 5, 100, $5)`, tenant, variant, loc.ID(), lot, now)

		f.receive(t, tenant, variant, loc.ID(), "2", "100")

		require.Equal(t, 2, f.levelRows(t, tenant, variant, loc.ID()))
		var lotQty decimal.Decimal
		require.NoError(t, f.db.WithTenantTx(t.Context(), tenant, func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(), "SELECT quantity FROM stock_levels WHERE lot_id = $1", lot).Scan(&lotQty)
		}))
		require.True(t, dec("5").Equal(lotQty))
		page, err := f.stock.ListLevels(t.Context(), tenant, domain.StockFilter{LocationID: loc.ID(), Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total, "the core lists only the stock without a lot")
		require.True(t, dec("2").Equal(page.Items[0].Quantity()))
	})
}
