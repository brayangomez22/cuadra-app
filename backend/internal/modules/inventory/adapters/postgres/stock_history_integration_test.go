//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStockRepositoryHasMovements(t *testing.T) {
	f := newFixture(t)
	tenant := f.createTenant(t)
	loc := f.createLocation(t, tenant, "Bodega")
	moved, still := f.createVariant(t, tenant), f.createVariant(t, tenant)
	f.receive(t, tenant, moved, loc.ID(), "10", "1000")

	t.Run("dice si alguna de las variantes tiene movimientos", func(t *testing.T) {
		got, err := f.stock.HasMovements(t.Context(), tenant, []uuid.UUID{still, moved})
		require.NoError(t, err)
		require.True(t, got)

		got, err = f.stock.HasMovements(t.Context(), tenant, []uuid.UUID{still})
		require.NoError(t, err)
		require.False(t, got)
	})

	t.Run("aislamiento: otra empresa no ve los movimientos", func(t *testing.T) {
		got, err := f.stock.HasMovements(t.Context(), f.createTenant(t), []uuid.UUID{moved})
		require.NoError(t, err)
		require.False(t, got)
	})
}
