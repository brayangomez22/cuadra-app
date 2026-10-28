package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestStockLevelReceiveInitial(t *testing.T) {
	cases := []struct {
		name, qty, cost, avg string
		from                 *[2]string
	}{
		{"la entrada inicial queda como initial_in con su costo (metros de cable)", "150.5", "2300", "2300", nil},
		{"la entrada inicial cambia el costo promedio como una compra (pares de calzado)", "10", "60000", "55000", &[2]string{"10", "50000"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			level := emptyLevel(t)
			if tc.from != nil {
				level = levelWith(t, tc.from[0], tc.from[1])
			}
			m, err := level.ReceiveInitial(dec(tc.qty), dec(tc.cost), testInfo())
			require.NoError(t, err)
			require.Equal(t, domain.MovementInitialIn, m.Type())
			requireDec(t, tc.qty, m.Quantity())
			requireDec(t, tc.cost, m.UnitCost())
			requireDec(t, tc.avg, level.AverageCost())
		})
	}

	t.Run("rechaza una entrada inicial sin cantidad", func(t *testing.T) {
		_, err := emptyLevel(t).ReceiveInitial(dec("0"), dec("100"), testInfo())
		require.ErrorIs(t, err, domain.ErrInvalidQuantity)
	})

	t.Run("acepta initial_in y la referencia a un conteo al reconstruir", func(t *testing.T) {
		mt, err := domain.ParseMovementType("initial_in")
		require.NoError(t, err)
		require.Equal(t, domain.MovementInitialIn, mt)

		s := validMovementSnapshot()
		s.Type, s.Quantity, s.Reason = domain.MovementAdjustment, dec("-1"), "Conteo físico"
		s.Reference = &domain.DocumentRef{Kind: domain.DocumentStockCount, ID: newID()}
		_, err = domain.RehydrateStockMovement(s)
		require.NoError(t, err)
	})
}
