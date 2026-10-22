package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestToBaseUnit(t *testing.T) {
	tests := []struct {
		name              string
		qty, cost, factor string
		wantQty, wantCost string
	}{
		{"ferretería: 1 rollo de 100 m a 150.000 son 100 m a 1.500", "1", "150000", "100", "100", "1500"},
		{"ferretería: 3 cajas ×12 a 5.500 son 36 und", "3", "5500", "12", "36", "458.3333"},
		{"almacén de ropa: 2 paquetes de 3 camisetas a 60.000 son 6 und a 20.000", "2", "60000", "3", "6", "20000"},
		{"tienda: 500 g a 30 el gramo son 0,5 kg a 30.000", "500", "30", "0.001", "0.5", "30000"},
		{"redondea el costo half-up a 4 decimales: 100 ÷ 6", "1", "100", "6", "6", "16.6667"},
		{"factor no entero: 2 unidades de 2,5 son 5", "2", "1000", "2.5", "5", "400"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qty, cost, err := domain.ToBaseUnit(dec(tt.qty), dec(tt.cost), dec(tt.factor))
			require.NoError(t, err)
			require.Truef(t, dec(tt.wantQty).Equal(qty), "cantidad: want %s, got %s", tt.wantQty, qty)
			require.Truef(t, dec(tt.wantCost).Equal(cost), "costo: want %s, got %s", tt.wantCost, cost)
		})
	}

	t.Run("rechaza datos que no se pueden convertir", func(t *testing.T) {
		cases := []struct {
			name              string
			qty, cost, factor string
			want              error
		}{
			{"factor cero", "1", "100", "0", domain.ErrInvalidUnitFactor},
			{"factor negativo", "1", "100", "-12", domain.ErrInvalidUnitFactor},
			{"factor con más de 4 decimales", "1", "100", "0.00001", domain.ErrInvalidUnitFactor},
			{"cantidad base con más de 4 decimales (0,25 g de un producto por kilo)", "0.25", "30", "0.001", domain.ErrQuantityPrecision},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				_, _, err := domain.ToBaseUnit(dec(c.qty), dec(c.cost), dec(c.factor))
				require.ErrorIs(t, err, c.want)
			})
		}
	})
}
