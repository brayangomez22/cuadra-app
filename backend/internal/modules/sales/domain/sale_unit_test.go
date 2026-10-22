package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestSaleLineUnit(t *testing.T) {
	t.Run("una línea sin unidad se vende en la unidad base y descuenta su misma cantidad", func(t *testing.T) {
		s := newSale(t)
		addLine(t, s, line("2.5", "1800", "0.19", ""))

		l := firstLine(t, s)
		require.Equal(t, domain.BaseUnit, l.Unit())
		require.Empty(t, l.Unit().Code)
		requireDec(t, "1", l.Unit().Factor)
		requireDec(t, "2.5", l.BaseQuantity())
		requireDec(t, "4500", l.Gross())
	})

	tests := []struct {
		name         string
		unit         domain.LineUnit
		qty, price   string
		wantBase     string
		wantGross    string
		wantQuantity string
	}{
		{"ferretería: 1 caja ×12 al precio de la caja", domain.LineUnit{Code: "caja", Factor: dec("12")}, "1", "110000", "12", "110000", "1"},
		{"ferretería: 2,5 rollos de 100 m", domain.LineUnit{Code: "rollo", Factor: dec("100")}, "2.5", "180000", "250", "450000", "2.5"},
		{"almacén de ropa: 2 paquetes de 3 camisetas", domain.LineUnit{Code: "paquete", Factor: dec("3")}, "2", "75000", "6", "150000", "2"},
		{"tienda: 250 g de un producto que se cuenta por kilo", domain.LineUnit{Code: "g", Factor: dec("0.001")}, "250", "12.5", "0.25", "3125", "250"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newSale(t)
			in := line(tt.qty, tt.price, "0.19", "")
			in.Unit = tt.unit
			addLine(t, s, in)

			l := firstLine(t, s)
			require.Equal(t, tt.unit.Code, l.Unit().Code)
			requireDec(t, tt.unit.Factor.String(), l.Unit().Factor)
			requireDec(t, tt.wantQuantity, l.Quantity())
			requireDec(t, tt.price, l.UnitPrice())
			requireDec(t, tt.wantBase, l.BaseQuantity())
			requireDec(t, tt.wantGross, l.Gross())
		})
	}

	t.Run("rechaza una línea con unidad inválida", func(t *testing.T) {
		cases := []struct {
			name string
			qty  string
			unit domain.LineUnit
			want error
		}{
			{"cantidad base con más de 4 decimales (0,25 g de un producto por kilo)", "0.25", domain.LineUnit{Code: "g", Factor: dec("0.001")}, domain.ErrQuantityPrecision},
			{"unidad alternativa con factor cero", "1", domain.LineUnit{Code: "caja", Factor: dec("0")}, domain.ErrInvalidUnitFactor},
			{"unidad alternativa con factor negativo", "1", domain.LineUnit{Code: "caja", Factor: dec("-12")}, domain.ErrInvalidUnitFactor},
			{"factor con más de 4 decimales", "1", domain.LineUnit{Code: "g", Factor: dec("0.00001")}, domain.ErrInvalidUnitFactor},
			{"unidad base con factor distinto de 1", "1", domain.LineUnit{Factor: dec("12")}, domain.ErrInvalidUnitFactor},
			{"código de unidad demasiado largo", "1", domain.LineUnit{Code: "unidad-con-un-codigo-larguisimo", Factor: dec("2")}, domain.ErrInvalidUnitCode},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				s := newSale(t)
				in := line(c.qty, "1000", "0.19", "")
				in.Unit = c.unit
				err := s.AddLine(in, domain.NoDiscountLimit)
				require.ErrorIs(t, err, c.want)
				require.Empty(t, s.Lines())
			})
		}
	})

	t.Run("reconstruye una venta guardada con la unidad de cada línea", func(t *testing.T) {
		orig := newSale(t)
		in := line("1", "110000", "0.19", "")
		in.Unit = domain.LineUnit{Code: "caja", Factor: dec("12")}
		addLine(t, orig, in)
		addLine(t, orig, line("2.5", "1800", "0.19", ""))
		require.NoError(t, orig.Confirm([]domain.Payment{payment(t, domain.PaymentCash, "200000")}, testNow))
		require.NoError(t, orig.AssignNumber(1))

		got, err := domain.RehydrateSale(snapshotOf(orig))
		require.NoError(t, err)
		require.Equal(t, orig.Lines(), got.Lines())
		requireDec(t, "12", got.Lines()[0].BaseQuantity())
		require.Equal(t, domain.BaseUnit, got.Lines()[1].Unit())
	})
}
