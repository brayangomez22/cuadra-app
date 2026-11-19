package domain_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestProfit(t *testing.T) {
	tests := []struct {
		name                   string
		revenue, cost          string
		wantAmount, wantMargin string
		wantMarginOK           bool
	}{
		{"ferretería: cemento vendido sobre su costo", "30000", "25000", "5000", "0.1667", true},
		{"almacén de ropa: camiseta con margen de 40 %", "50000", "30000", "20000", "0.4", true},
		{"una venta bajo el costo tiene utilidad y margen negativos", "20000", "25000", "-5000", "-0.25", true},
		{"redondea la utilidad a 2 decimales con half-up", "100.005", "0", "100.01", "1", true},
		{"una venta con costo cero tiene margen del 100 %", "15000", "0", "15000", "1", true},
		{"sin ingreso no hay margen (no divide por cero)", "0", "1000", "-1000", "0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := domain.Profit{Revenue: dec(tt.revenue), Cost: dec(tt.cost)}
			require.True(t, dec(tt.wantAmount).Equal(p.Amount()), "utilidad %s", p.Amount())
			m, ok := p.Margin()
			require.Equal(t, tt.wantMarginOK, ok)
			require.True(t, dec(tt.wantMargin).Equal(m), "margen %s", m)
		})
	}

	t.Run("suma ingresos y costos", func(t *testing.T) {
		p := domain.Profit{Revenue: dec("100"), Cost: dec("60")}.Add(domain.Profit{Revenue: dec("50"), Cost: dec("20")})
		require.True(t, dec("150").Equal(p.Revenue))
		require.True(t, dec("80").Equal(p.Cost))
	})
}

func TestRate(t *testing.T) {
	r, ok := domain.Rate(dec("1"), dec("3"))
	require.True(t, ok)
	require.True(t, dec("0.3333").Equal(r), r.String())
	_, ok = domain.Rate(dec("1"), decimal.Zero)
	require.False(t, ok)
	require.True(t, dec("12.35").Equal(domain.Money(dec("12.345"))))
}
