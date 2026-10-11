package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestParsePaymentMethod(t *testing.T) {
	t.Run("acepta los cinco medios de pago", func(t *testing.T) {
		for _, in := range []string{"cash", "card", "nequi", "daviplata", "transfer"} {
			got, err := domain.ParsePaymentMethod(in)
			require.NoError(t, err, in)
			require.Equal(t, in, got.String())
		}
	})

	t.Run("rechaza método de pago desconocido", func(t *testing.T) {
		for _, in := range []string{"", "Cash", "bitcoin", " card"} {
			_, err := domain.ParsePaymentMethod(in)
			require.ErrorIs(t, err, domain.ErrInvalidPaymentMethod, in)
		}
	})
}

func TestNewPayment(t *testing.T) {
	t.Run("crea un pago", func(t *testing.T) {
		p, err := domain.NewPayment(domain.PaymentNequi, dec("15000.50"))
		require.NoError(t, err)
		require.Equal(t, domain.PaymentNequi, p.Method())
		requireDec(t, "15000.50", p.Amount())
	})

	t.Run("rechaza método de pago desconocido", func(t *testing.T) {
		_, err := domain.NewPayment(domain.PaymentMethod("bitcoin"), dec("100"))
		require.ErrorIs(t, err, domain.ErrInvalidPaymentMethod)
	})

	t.Run("rechaza monto no positivo, con más de 2 decimales o que no cabe", func(t *testing.T) {
		for _, in := range []string{"0", "-100", "100.001", "100000000000000"} {
			_, err := domain.NewPayment(domain.PaymentCash, dec(in))
			require.ErrorIs(t, err, domain.ErrInvalidPaymentAmount, in)
		}
	})
}
