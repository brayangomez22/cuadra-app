package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
)

func TestStatement(t *testing.T) {
	t.Run("ordena del más reciente al más antiguo con el saldo que dejó cada movimiento", func(t *testing.T) {
		day := func(d int) time.Time { return time.Date(2026, 10, d, 15, 0, 0, 0, time.UTC) }
		got := domain.Statement([]domain.Movement{
			{Kind: domain.MovementPayment, At: day(5), Amount: dec("30000")},
			{Kind: domain.MovementCharge, At: day(1), Amount: dec("100000")},      // ferretería
			{Kind: domain.MovementCharge, At: day(3), Amount: dec("20000")},       // almacén de ropa
			{Kind: domain.MovementCancellation, At: day(4), Amount: dec("20000")}, // se anuló la del día 3
		})
		require.Len(t, got, 4)
		wantKinds := []domain.MovementKind{domain.MovementPayment, domain.MovementCancellation, domain.MovementCharge, domain.MovementCharge}
		wantBalances := []string{"70000", "100000", "120000", "100000"}
		for i := range got {
			require.Equal(t, wantKinds[i], got[i].Kind, i)
			require.True(t, dec(wantBalances[i]).Equal(got[i].Balance), "%d: %s", i, got[i].Balance)
		}
	})

	t.Run("a la misma hora el cargo va antes que el abono", func(t *testing.T) {
		at := time.Date(2026, 10, 1, 15, 0, 0, 0, time.UTC)
		got := domain.Statement([]domain.Movement{
			{Kind: domain.MovementPayment, At: at, Amount: dec("5000")},
			{Kind: domain.MovementCharge, At: at, Amount: dec("5000")},
		})
		require.Equal(t, domain.MovementPayment, got[0].Kind)
		require.True(t, got[0].Balance.IsZero())
	})
}
