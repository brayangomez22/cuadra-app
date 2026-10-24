//go:build integration

package postgres_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestStockRepositoryApply(t *testing.T) {
	f := newFixture(t)

	t.Run("dos salidas simultáneas por el total del stock: solo una pasa", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")
		f.receive(t, tenant, variant, loc.ID(), "10", "1000")

		const workers = 8
		start := make(chan struct{})
		errs := make([]error, workers)
		var wg sync.WaitGroup
		for i := range workers {
			wg.Go(func() {
				<-start
				errs[i] = f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{loc.ID()},
					func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
						m, err := levels[0].Issue(dec("10"), domain.StockPolicy{}, info(newID()))
						return []*domain.StockMovement{m}, err
					})
			})
		}
		close(start)
		wg.Wait()

		var ok, insufficient int
		for _, err := range errs {
			switch {
			case err == nil:
				ok++
			case errors.Is(err, domain.ErrInsufficientStock):
				insufficient++
			default:
				t.Fatalf("unexpected error: %v", err)
			}
		}
		require.Equal(t, 1, ok)
		require.Equal(t, workers-1, insufficient)
		require.True(t, f.level(t, tenant, variant, loc.ID()).IsZero())
		kardex, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{VariantID: variant, Limit: 100})
		require.NoError(t, err)
		require.Equal(t, 2, kardex.Total)
	})

	t.Run("dos traslados opuestos simultáneos no se bloquean entre sí", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		a := f.createLocation(t, tenant, "Sede A")
		b := f.createLocation(t, tenant, "Sede B")
		f.receive(t, tenant, variant, a.ID(), "100", "1000")
		f.receive(t, tenant, variant, b.ID(), "100", "1000")

		transfer := func(from, to uuid.UUID) error {
			return f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{from, to},
				func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
					out, in, err := domain.Transfer(levels[0], levels[1], dec("1"), domain.StockPolicy{}, newID(), now)
					return []*domain.StockMovement{out, in}, err
				})
		}
		const rounds = 20
		start := make(chan struct{})
		errs := make(chan error, 2*rounds)
		var wg sync.WaitGroup
		for range rounds {
			wg.Go(func() { <-start; errs <- transfer(a.ID(), b.ID()) })
			wg.Go(func() { <-start; errs <- transfer(b.ID(), a.ID()) })
		}
		close(start)
		wg.Wait()
		close(errs)

		for err := range errs {
			require.NoError(t, err)
		}
		require.True(t, dec("100").Equal(f.level(t, tenant, variant, a.ID())))
		require.True(t, dec("100").Equal(f.level(t, tenant, variant, b.ID())))
	})

	t.Run("una operación fallida no deja niveles ni movimientos", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Bodega")

		err := f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{loc.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Issue(dec("1"), domain.StockPolicy{}, info(newID()))
				return []*domain.StockMovement{m}, err
			})
		require.ErrorIs(t, err, domain.ErrInsufficientStock)

		levels, err := f.stock.ListLevels(t.Context(), tenant, domain.StockFilter{LocationID: loc.ID(), Limit: 10})
		require.NoError(t, err)
		require.Zero(t, levels.Total)
		kardex, err := f.stock.Kardex(t.Context(), tenant, domain.KardexFilter{VariantID: variant, Limit: 10})
		require.NoError(t, err)
		require.Zero(t, kardex.Total)
	})

	t.Run("una variante inexistente da ErrVariantNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		loc := f.createLocation(t, tenant, "Bodega")

		err := f.stock.Apply(t.Context(), tenant, newID(), []uuid.UUID{loc.ID()},
			func([]*domain.StockLevel) ([]*domain.StockMovement, error) { return nil, nil })
		require.ErrorIs(t, err, domain.ErrVariantNotFound)
	})

	t.Run("una sede inexistente da ErrLocationNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)

		err := f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{newID()},
			func([]*domain.StockLevel) ([]*domain.StockMovement, error) { return nil, nil })
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})

	t.Run("entrega los niveles en el orden de las sedes pedidas", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		first := f.createLocation(t, tenant, "Primera")
		second := f.createLocation(t, tenant, "Segunda")

		err := f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{second.ID(), first.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				require.Equal(t, second.ID(), levels[0].LocationID())
				require.Equal(t, first.ID(), levels[1].LocationID())
				return nil, nil
			})
		require.NoError(t, err)
	})
}

func TestMovementsAreImmutable(t *testing.T) {
	f := newFixture(t)
	tenant := f.createTenant(t)
	variant := f.createVariant(t, tenant)
	loc := f.createLocation(t, tenant, "Bodega")
	f.receive(t, tenant, variant, loc.ID(), "10", "1000")

	for _, sql := range []string{
		"UPDATE stock_movements SET quantity = 99",
		"DELETE FROM stock_movements",
	} {
		t.Run("app_user no puede modificar ni borrar movimientos: "+sql, func(t *testing.T) {
			err := f.db.WithTenantTx(t.Context(), tenant, func(tx pgx.Tx) error {
				_, err := tx.Exec(t.Context(), sql)
				return err
			})
			var pgErr *pgconn.PgError
			require.ErrorAs(t, err, &pgErr)
			require.Equal(t, "42501", pgErr.Code, "insufficient_privilege")
		})
	}
}
