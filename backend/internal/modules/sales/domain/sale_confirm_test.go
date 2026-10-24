package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestDiscountLimit(t *testing.T) {
	tenPercent, err := domain.NewDiscountLimit(dec("0.10"))
	require.NoError(t, err)

	tests := []struct {
		name     string
		limit    domain.DiscountLimit
		discount string
		want     error
	}{
		{"rechaza un descuento del cajero mayor a su tope", tenPercent, "2000.01", domain.ErrDiscountExceedsLimit},
		{"acepta un descuento igual al tope", tenPercent, "2000", nil},
		{"sin tope se permite descontar hasta el bruto de la línea", domain.NoDiscountLimit, "20000", nil},
		{"el tope en cero no permite descuentos", domain.DiscountLimit{}, "0.01", domain.ErrDiscountExceedsLimit},
		{"el tope en cero permite vender sin descuento", domain.DiscountLimit{}, "0", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newSale(t)
			err := s.AddLine(line("2", "10000", "0.19", tt.discount), tt.limit) // bruto 20000
			if tt.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.want)
			require.Empty(t, s.Lines())
		})
	}

	t.Run("rechaza topes fuera de 0 a 1 o con más de 4 decimales", func(t *testing.T) {
		for _, in := range []string{"-0.01", "1.0001", "0.00001"} {
			_, err := domain.NewDiscountLimit(dec(in))
			require.ErrorIs(t, err, domain.ErrInvalidDiscountLimit, in)
		}
	})

	t.Run("expone el tope", func(t *testing.T) {
		requireDec(t, "0.10", tenPercent.Max())
		require.False(t, tenPercent.Unlimited())
		require.True(t, domain.NoDiscountLimit.Unlimited())
	})
}

func TestSaleConfirm(t *testing.T) {
	t.Run("calcula el cambio con pago mixto efectivo + Nequi", func(t *testing.T) {
		s := mixedSale(t) // total 31710.79
		err := s.Confirm([]domain.Payment{
			payment(t, domain.PaymentNequi, "15000"),
			payment(t, domain.PaymentCash, "20000"),
		}, testNow)
		require.NoError(t, err)

		require.Equal(t, domain.SaleConfirmed, s.Status())
		requireDec(t, "35000", s.Paid())
		requireDec(t, "3289.21", s.Change())
		require.Len(t, s.Payments(), 2)
		require.NotNil(t, s.ConfirmedAt())
		require.Equal(t, testNow.UTC(), *s.ConfirmedAt())
	})

	t.Run("confirma con pago exacto con tarjeta y sin cambio", func(t *testing.T) {
		s := mixedSale(t)
		require.NoError(t, s.Confirm([]domain.Payment{payment(t, domain.PaymentCard, "31710.79")}, testNow))
		require.True(t, s.Change().IsZero())
	})

	t.Run("rechaza pago insuficiente", func(t *testing.T) {
		s := mixedSale(t)
		err := s.Confirm([]domain.Payment{
			payment(t, domain.PaymentCash, "30000"),
			payment(t, domain.PaymentDaviplata, "1710.78"),
		}, testNow)
		require.ErrorIs(t, err, domain.ErrInsufficientPayment)
		require.Equal(t, domain.SaleDraft, s.Status())
		require.Empty(t, s.Payments())
	})

	t.Run("rechaza confirmar sin pagos si el total es mayor a cero", func(t *testing.T) {
		require.ErrorIs(t, mixedSale(t).Confirm(nil, testNow), domain.ErrInsufficientPayment)
	})

	t.Run("rechaza que los pagos que no son en efectivo superen el total", func(t *testing.T) {
		s := mixedSale(t)
		err := s.Confirm([]domain.Payment{
			payment(t, domain.PaymentCard, "20000"),
			payment(t, domain.PaymentTransfer, "12000"),
		}, testNow)
		require.ErrorIs(t, err, domain.ErrNonCashOverpayment)
	})

	t.Run("rechaza confirmar una venta sin líneas", func(t *testing.T) {
		s := newSale(t)
		require.ErrorIs(t, s.Confirm(nil, testNow), domain.ErrSaleWithoutLines)
	})

	t.Run("una venta confirmada no acepta líneas ni se confirma dos veces", func(t *testing.T) {
		s := mixedSale(t)
		pay := []domain.Payment{payment(t, domain.PaymentCash, "40000")}
		require.NoError(t, s.Confirm(pay, testNow))

		require.ErrorIs(t, s.AddLine(line("1", "100", "0", ""), domain.NoDiscountLimit), domain.ErrSaleNotDraft)
		require.ErrorIs(t, s.Confirm(pay, testNow), domain.ErrSaleNotDraft)
		require.Len(t, s.Lines(), 4)
	})

	t.Run("una venta en borrador no tiene fecha de confirmación", func(t *testing.T) {
		require.Nil(t, newSale(t).ConfirmedAt())
	})
}
