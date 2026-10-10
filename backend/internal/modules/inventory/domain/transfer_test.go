package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// siblingLevel returns a level of the same tenant and product as base, at
// another location.
func siblingLevel(t *testing.T, base *domain.StockLevel, qty, avgCost string) *domain.StockLevel {
	t.Helper()
	level, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
		TenantID:    base.TenantID(),
		ProductID:   base.ProductID(),
		LocationID:  newID(),
		Quantity:    dec(qty),
		AverageCost: dec(avgCost),
	})
	require.NoError(t, err)
	return level
}

func TestTransfer(t *testing.T) {
	t.Run("un traslado genera dos movimientos que suman cero", func(t *testing.T) {
		from := levelWith(t, "10", "1000")
		to := siblingLevel(t, from, "0", "0")
		userID := newID()

		out, in, err := domain.Transfer(from, to, dec("2.5"), strict, userID, testNow)
		require.NoError(t, err)
		require.Equal(t, domain.MovementTransferOut, out.Type())
		require.Equal(t, domain.MovementTransferIn, in.Type())
		require.Equal(t, from.LocationID(), out.LocationID())
		require.Equal(t, to.LocationID(), in.LocationID())
		requireDec(t, "-2.5", out.Quantity())
		requireDec(t, "2.5", in.Quantity())
		requireDec(t, "0", out.Quantity().Add(in.Quantity()))
		require.Equal(t, userID, out.UserID())
		require.Equal(t, userID, in.UserID())
		requireDec(t, "7.5", from.Quantity())
		requireDec(t, "2.5", to.Quantity())
		require.NotEqual(t, out.ID(), in.ID())
	})

	t.Run("el traslado lleva el costo promedio del origen y recalcula el del destino", func(t *testing.T) {
		from := levelWith(t, "10", "1200")
		to := siblingLevel(t, from, "10", "1000")

		out, in, err := domain.Transfer(from, to, dec("10"), strict, newID(), testNow)
		require.NoError(t, err)
		requireDec(t, "1200", out.UnitCost())
		requireDec(t, "1200", in.UnitCost())
		requireDec(t, "1200", from.AverageCost())
		requireDec(t, "1100", to.AverageCost())
		requireDec(t, "1100", in.AverageCostAfter())
	})

	t.Run("ambos movimientos comparten la referencia del traslado", func(t *testing.T) {
		from := levelWith(t, "10", "1000")
		to := siblingLevel(t, from, "0", "0")

		out, in, err := domain.Transfer(from, to, dec("1"), strict, newID(), testNow)
		require.NoError(t, err)
		require.NotNil(t, out.Reference())
		require.Equal(t, domain.DocumentTransfer, out.Reference().Kind)
		require.Equal(t, 7, int(out.Reference().ID.Version()))
		require.Equal(t, out.Reference(), in.Reference())
	})

	t.Run("un traslado sin stock suficiente falla y no cambia ningún nivel", func(t *testing.T) {
		from := levelWith(t, "2", "1000")
		to := siblingLevel(t, from, "5", "900")

		_, _, err := domain.Transfer(from, to, dec("3"), strict, newID(), testNow)
		require.ErrorIs(t, err, domain.ErrInsufficientStock)
		requireDec(t, "2", from.Quantity())
		requireDec(t, "5", to.Quantity())
		requireDec(t, "900", to.AverageCost())
	})

	t.Run("permite traslado sin stock si la política admite negativos", func(t *testing.T) {
		from := levelWith(t, "2", "1000")
		to := siblingLevel(t, from, "0", "0")

		_, _, err := domain.Transfer(from, to, dec("3"), allowShort, newID(), testNow)
		require.NoError(t, err)
		requireDec(t, "-1", from.Quantity())
		requireDec(t, "3", to.Quantity())
	})

	t.Run("rechaza traslado a la misma sede", func(t *testing.T) {
		from := levelWith(t, "10", "1000")
		_, _, err := domain.Transfer(from, from, dec("1"), strict, newID(), testNow)
		require.ErrorIs(t, err, domain.ErrSameLocationTransfer)
		requireDec(t, "10", from.Quantity())
	})

	t.Run("rechaza traslado entre productos o tenants distintos", func(t *testing.T) {
		from := levelWith(t, "10", "1000")
		otherProduct, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
			TenantID: from.TenantID(), ProductID: newID(), LocationID: newID(),
			Quantity: dec("0"), AverageCost: dec("0"),
		})
		require.NoError(t, err)
		otherTenant, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
			TenantID: newID(), ProductID: from.ProductID(), LocationID: newID(),
			Quantity: dec("0"), AverageCost: dec("0"),
		})
		require.NoError(t, err)

		for _, to := range []*domain.StockLevel{otherProduct, otherTenant} {
			_, _, err := domain.Transfer(from, to, dec("1"), strict, newID(), testNow)
			require.ErrorIs(t, err, domain.ErrTransferMismatch)
		}
		requireDec(t, "10", from.Quantity())
	})

	t.Run("rechaza cantidad inválida o usuario vacío", func(t *testing.T) {
		from := levelWith(t, "10", "1000")
		to := siblingLevel(t, from, "0", "0")

		_, _, err := domain.Transfer(from, to, dec("0"), strict, newID(), testNow)
		require.ErrorIs(t, err, domain.ErrInvalidQuantity)
		_, _, err = domain.Transfer(from, to, dec("1"), strict, uuid.Nil, testNow)
		require.ErrorIs(t, err, domain.ErrInvalidUserID)
		requireDec(t, "10", from.Quantity())
		requireDec(t, "0", to.Quantity())
	})
}
