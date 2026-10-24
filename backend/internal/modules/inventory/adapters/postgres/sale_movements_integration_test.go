//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestStockRepositorySaleMovements(t *testing.T) {
	f := newFixture(t)

	sell := func(t *testing.T, tenantID, variantID, locationID uuid.UUID, ref domain.DocumentRef, qty string) {
		t.Helper()
		err := f.stock.Apply(t.Context(), tenantID, variantID, []uuid.UUID{locationID},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				i := info(newID())
				i.Reference = &ref
				m, err := levels[0].Issue(dec(qty), domain.StockPolicy{}, i)
				return []*domain.StockMovement{m}, err
			})
		require.NoError(t, err)
	}

	t.Run("lista los movimientos de una venta y guarda su sale_void", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Sede")
		f.receive(t, tenant, variant, loc.ID(), "10", "1000")
		sale := domain.DocumentRef{Kind: domain.DocumentSale, ID: newID()}
		sell(t, tenant, variant, loc.ID(), sale, "3")
		sell(t, tenant, variant, loc.ID(), domain.DocumentRef{Kind: domain.DocumentSale, ID: newID()}, "1")

		got, err := f.stock.MovementsByReference(t.Context(), tenant, sale)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, domain.MovementSaleOut, got[0].Type())
		require.True(t, dec("-3").Equal(got[0].Quantity()))

		err = f.stock.Apply(t.Context(), tenant, variant, []uuid.UUID{loc.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				i := info(newID())
				i.Reference = &sale
				m, err := levels[0].ReturnSale(dec("3"), got[0].UnitCost(), i)
				return []*domain.StockMovement{m}, err
			})
		require.NoError(t, err)
		require.True(t, dec("9").Equal(f.level(t, tenant, variant, loc.ID())))

		got, err = f.stock.MovementsByReference(t.Context(), tenant, sale)
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, domain.MovementSaleVoid, got[1].Type())
	})

	t.Run("la base de datos rechaza un sale_void sin referencia de venta", func(t *testing.T) {
		tenant := f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Sede")
		err := f.db.WithTenantTx(t.Context(), tenant, func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), `INSERT INTO stock_movements (id, tenant_id, variant_id, location_id, type, quantity,
				unit_cost, balance_after, average_cost_after, reason, user_id, occurred_at)
				VALUES ($1, $2, $3, $4, 'sale_void', 1, 0, 1, 0, '', $5, $6)`, newID(), tenant, variant, loc.ID(), newID(), now)
			return err
		})
		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		require.Equal(t, "stock_movements_sale_void_reference", pgErr.ConstraintName)
	})

	t.Run("aislamiento: otro tenant no ve los movimientos de la venta", func(t *testing.T) {
		tenant, other := f.createTenant(t), f.createTenant(t)
		variant := f.createVariant(t, tenant)
		loc := f.createLocation(t, tenant, "Sede")
		f.receive(t, tenant, variant, loc.ID(), "10", "1000")
		sale := domain.DocumentRef{Kind: domain.DocumentSale, ID: newID()}
		sell(t, tenant, variant, loc.ID(), sale, "3")

		got, err := f.stock.MovementsByReference(t.Context(), other, sale)
		require.NoError(t, err)
		require.Empty(t, got)
	})
}
