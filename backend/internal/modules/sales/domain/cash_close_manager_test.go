package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestCashSessionCloseByManager(t *testing.T) {
	t.Run("quien administra la caja cierra la que otro dejó abierta y queda como quien la cerró", func(t *testing.T) {
		s := openSession(t, "50000")
		manager := newID()
		in := closeInput(s, count(t, "50000", 1), activity("0", "0", "0", "0"))
		in.UserID, in.ManagesCash = manager, true

		require.NoError(t, s.Close(in))

		require.Equal(t, domain.CashSessionClosed, s.Status())
		require.Equal(t, manager, s.Closing().ClosedBy)
		require.NotEqual(t, s.UserID(), s.Closing().ClosedBy)
	})

	t.Run("la caja cerrada por quien la administra se reconstruye igual", func(t *testing.T) {
		s := openSession(t, "50000")
		in := closeInput(s, count(t, "50000", 1), activity("0", "0", "0", "0"))
		in.UserID, in.ManagesCash = newID(), true
		require.NoError(t, s.Close(in))

		back, err := domain.RehydrateCashSession(s.Snapshot())
		require.NoError(t, err)
		require.Equal(t, s.Closing().ClosedBy, back.Closing().ClosedBy)
	})

	t.Run("sin el permiso de manejar la caja, otro usuario no la cierra", func(t *testing.T) {
		s := openSession(t, "0")
		in := closeInput(s, count(t), activity("0", "0", "0", "0"))
		in.UserID = newID()
		require.ErrorIs(t, s.Close(in), domain.ErrCashSessionNotOwner)
	})

	t.Run("el permiso no deja registrar movimientos en la caja de otro", func(t *testing.T) {
		s := openSession(t, "0")
		in := movement(s, domain.CashIn, "1000", "Base extra")
		in.UserID = newID()
		_, err := s.RecordMovement(in)
		require.ErrorIs(t, err, domain.ErrCashSessionNotOwner)
	})
}

func TestCashActivitySalesTotal(t *testing.T) {
	tests := []struct {
		name     string
		byMethod []domain.MethodTotal
		want     string
	}{
		{name: "sin ventas es cero", want: "0"},
		{
			name: "suma lo que cobró cada medio en una ferretería, el efectivo ya sin el cambio",
			byMethod: []domain.MethodTotal{
				{Method: domain.PaymentCash, Amount: dec("230000")},
				{Method: domain.PaymentCard, Amount: dec("125500.50")},
			},
			want: "355500.50",
		},
		{
			name: "incluye lo vendido a crédito en un almacén de ropa",
			byMethod: []domain.MethodTotal{
				{Method: domain.PaymentNequi, Amount: dec("89900")},
				{Method: domain.PaymentCredit, Amount: dec("150000")},
			},
			want: "239900",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireDec(t, tt.want, domain.CashActivity{ByMethod: tt.byMethod}.SalesTotal())
		})
	}
}
