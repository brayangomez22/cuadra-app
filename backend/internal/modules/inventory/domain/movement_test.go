package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestParseMovementType(t *testing.T) {
	t.Run("acepta los cinco tipos de movimiento", func(t *testing.T) {
		for _, s := range []string{"purchase_in", "sale_out", "adjustment", "transfer_out", "transfer_in"} {
			mt, err := domain.ParseMovementType(s)
			require.NoError(t, err, s)
			require.Equal(t, s, string(mt))
		}
	})

	t.Run("rechaza un tipo desconocido", func(t *testing.T) {
		for _, s := range []string{"", "PURCHASE_IN", "return_in"} {
			_, err := domain.ParseMovementType(s)
			require.ErrorIs(t, err, domain.ErrInvalidMovementType, s)
		}
	})
}

func validMovementSnapshot() domain.StockMovementSnapshot {
	return domain.StockMovementSnapshot{
		ID:               newID(),
		TenantID:         newID(),
		ProductID:        newID(),
		LocationID:       newID(),
		Type:             domain.MovementSaleOut,
		Quantity:         dec("-2.5"),
		UnitCost:         dec("1100"),
		BalanceAfter:     dec("7.5"),
		AverageCostAfter: dec("1100"),
		Reference:        &domain.DocumentRef{Kind: domain.DocumentSale, ID: newID()},
		UserID:           newID(),
		OccurredAt:       testNow,
	}
}

func TestRehydrateStockMovement(t *testing.T) {
	t.Run("rehidrata un movimiento guardado", func(t *testing.T) {
		s := validMovementSnapshot()
		m, err := domain.RehydrateStockMovement(s)
		require.NoError(t, err)
		require.Equal(t, s.ID, m.ID())
		require.Equal(t, s.TenantID, m.TenantID())
		require.Equal(t, s.ProductID, m.ProductID())
		require.Equal(t, s.LocationID, m.LocationID())
		require.Equal(t, s.Type, m.Type())
		requireDec(t, "-2.5", m.Quantity())
		requireDec(t, "1100", m.UnitCost())
		requireDec(t, "7.5", m.BalanceAfter())
		requireDec(t, "1100", m.AverageCostAfter())
		require.Equal(t, s.Reference, m.Reference())
		require.Equal(t, s.UserID, m.UserID())
		require.Equal(t, testNow.UTC(), m.OccurredAt())
	})

	t.Run("la referencia devuelta es una copia", func(t *testing.T) {
		s := validMovementSnapshot()
		m, err := domain.RehydrateStockMovement(s)
		require.NoError(t, err)
		s.Reference.ID = newID()
		m.Reference().ID = newID()
		require.NotEqual(t, s.Reference.ID, m.Reference().ID)
		require.NotEqual(t, uuid.Nil, m.Reference().ID)
	})

	t.Run("rechaza datos corruptos", func(t *testing.T) {
		cases := map[string]struct {
			mutate func(*domain.StockMovementSnapshot)
			want   error
		}{
			"id vacío":       {func(s *domain.StockMovementSnapshot) { s.ID = uuid.Nil }, domain.ErrInvalidMovementID},
			"tenant vacío":   {func(s *domain.StockMovementSnapshot) { s.TenantID = uuid.Nil }, domain.ErrInvalidTenantID},
			"producto vacío": {func(s *domain.StockMovementSnapshot) { s.ProductID = uuid.Nil }, domain.ErrInvalidProductID},
			"sede vacía":     {func(s *domain.StockMovementSnapshot) { s.LocationID = uuid.Nil }, domain.ErrInvalidLocationID},
			"usuario vacío":  {func(s *domain.StockMovementSnapshot) { s.UserID = uuid.Nil }, domain.ErrInvalidUserID},
			"tipo inválido":  {func(s *domain.StockMovementSnapshot) { s.Type = "robo" }, domain.ErrInvalidMovementType},
			"cantidad cero":  {func(s *domain.StockMovementSnapshot) { s.Quantity = dec("0") }, domain.ErrInvalidMovement},
			"signo que no coincide con el tipo": {
				func(s *domain.StockMovementSnapshot) { s.Quantity = dec("2.5") }, domain.ErrInvalidMovement,
			},
			"costo unitario negativo": {
				func(s *domain.StockMovementSnapshot) { s.UnitCost = dec("-1") }, domain.ErrInvalidMovement,
			},
			"saldo con 5 decimales": {
				func(s *domain.StockMovementSnapshot) { s.BalanceAfter = dec("1.00001") }, domain.ErrInvalidMovement,
			},
			"ajuste sin motivo": {
				func(s *domain.StockMovementSnapshot) { s.Type, s.Reference = domain.MovementAdjustment, nil },
				domain.ErrAdjustmentReasonRequired,
			},
			"traslado sin referencia de traslado": {
				func(s *domain.StockMovementSnapshot) { s.Type = domain.MovementTransferOut },
				domain.ErrInvalidReference,
			},
		}
		for name, tc := range cases {
			s := validMovementSnapshot()
			tc.mutate(&s)
			_, err := domain.RehydrateStockMovement(s)
			require.ErrorIs(t, err, tc.want, name)
		}
	})
}
