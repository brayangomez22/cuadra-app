package httpx_test

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

func TestFormatMoney(t *testing.T) {
	cases := []struct{ in, want string }{
		{"10000", "10000.00"},
		{"12500.5", "12500.50"},
		{"8333.333", "8333.333"},
		{"8403.3613", "8403.3613"},
		{"0.00001", "0.0000"},
		{"-1500", "-1500.00"},
	}
	for _, c := range cases {
		t.Run("escribe "+c.in+" como "+c.want, func(t *testing.T) {
			require.Equal(t, c.want, httpx.FormatMoney(decimal.RequireFromString(c.in)))
		})
	}
}

func TestFormatDecimal(t *testing.T) {
	t.Run("escribe cantidades y tasas sin ceros a la derecha", func(t *testing.T) {
		require.Equal(t, "2.5", httpx.FormatDecimal(decimal.RequireFromString("2.5000")))
		require.Equal(t, "0.19", httpx.FormatDecimal(decimal.RequireFromString("0.1900")))
		require.Equal(t, "-3", httpx.FormatDecimal(decimal.RequireFromString("-3")))
	})
}

func TestParseDecimal(t *testing.T) {
	errInvalid := errors.New("cantidad inválida")

	t.Run("lee un decimal del contrato", func(t *testing.T) {
		d, err := httpx.ParseDecimal("0.75", errInvalid)

		require.NoError(t, err)
		require.True(t, decimal.RequireFromString("0.75").Equal(d))
	})

	t.Run("un valor que no es decimal devuelve el error del dominio", func(t *testing.T) {
		_, err := httpx.ParseDecimal("dos", errInvalid)

		require.ErrorIs(t, err, errInvalid)
	})
}
