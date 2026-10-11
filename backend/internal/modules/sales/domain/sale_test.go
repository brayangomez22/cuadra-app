package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

var testNow = time.Date(2026, 10, 10, 9, 30, 0, 0, time.FixedZone("COT", -5*3600))

// requireDec compares decimals by value, so 1100 equals 1100.00.
func requireDec(t *testing.T, want string, got decimal.Decimal) {
	t.Helper()
	require.Truef(t, dec(want).Equal(got), "want %s, got %s", want, got)
}

func newSale(t *testing.T) *domain.Sale {
	t.Helper()
	s, err := domain.NewSale(domain.NewSaleParams{
		ID: newID(), TenantID: newID(), LocationID: newID(), UserID: newID(), CreatedAt: testNow,
	})
	require.NoError(t, err)
	return s
}

// line is a line input of one product; discount is "0" if empty.
func line(qty, price, rate, discount string) domain.LineInput {
	if discount == "" {
		discount = "0"
	}
	return domain.LineInput{
		ProductID: newID(), ProductName: "Producto", Quantity: dec(qty),
		UnitPrice: dec(price), TaxRate: dec(rate), Discount: dec(discount),
	}
}

func addLine(t *testing.T, s *domain.Sale, in domain.LineInput) {
	t.Helper()
	require.NoError(t, s.AddLine(in, domain.NoDiscountLimit))
}

// firstLine returns the sale's first line, failing if it has none.
func firstLine(t *testing.T, s *domain.Sale) domain.SaleLine {
	t.Helper()
	require.NotEmpty(t, s.Lines())
	return s.Lines()[0]
}

func payment(t *testing.T, method domain.PaymentMethod, amount string) domain.Payment {
	t.Helper()
	p, err := domain.NewPayment(method, dec(amount))
	require.NoError(t, err)
	return p
}

// mixedSale has lines at 0%, 5% and 19% IVA; its total is 31710.79.
func mixedSale(t *testing.T) *domain.Sale {
	t.Helper()
	s := newSale(t)
	addLine(t, s, line("10", "250", "0.19", ""))      // base 2500, IVA 475
	addLine(t, s, line("2", "10000", "0.05", "1000")) // bruto 20000, base 19000, IVA 950
	addLine(t, s, line("1", "4500", "0", ""))         // base 4500, IVA 0
	addLine(t, s, line("3", "1200.50", "0.19", ""))   // base 3601.50, IVA 684.285 → 684.29
	return s
}

func TestNewSale(t *testing.T) {
	t.Run("crea una venta en borrador sin líneas ni cliente", func(t *testing.T) {
		p := domain.NewSaleParams{ID: newID(), TenantID: newID(), LocationID: newID(), UserID: newID(), CreatedAt: testNow}
		s, err := domain.NewSale(p)
		require.NoError(t, err)

		require.Equal(t, p.ID, s.ID())
		require.Equal(t, p.TenantID, s.TenantID())
		require.Equal(t, p.LocationID, s.LocationID())
		require.Equal(t, p.UserID, s.UserID())
		require.Nil(t, s.CustomerID())
		require.Equal(t, domain.SaleDraft, s.Status())
		require.Empty(t, s.Lines())
		require.Equal(t, testNow.UTC(), s.CreatedAt())
		require.Equal(t, time.UTC, s.CreatedAt().Location())
	})

	t.Run("guarda el cliente opcional", func(t *testing.T) {
		customer := newID()
		s, err := domain.NewSale(domain.NewSaleParams{
			ID: newID(), TenantID: newID(), LocationID: newID(), UserID: newID(), CustomerID: &customer, CreatedAt: testNow,
		})
		require.NoError(t, err)
		require.NotNil(t, s.CustomerID())
		require.Equal(t, customer, *s.CustomerID())
	})

	t.Run("rechaza ids vacíos", func(t *testing.T) {
		valid := func() domain.NewSaleParams {
			return domain.NewSaleParams{ID: newID(), TenantID: newID(), LocationID: newID(), UserID: newID(), CreatedAt: testNow}
		}
		nilCustomer := uuid.Nil
		tests := []struct {
			name   string
			mutate func(*domain.NewSaleParams)
			want   error
		}{
			{"sin id", func(p *domain.NewSaleParams) { p.ID = uuid.Nil }, domain.ErrInvalidSaleID},
			{"sin tenant", func(p *domain.NewSaleParams) { p.TenantID = uuid.Nil }, domain.ErrInvalidTenantID},
			{"sin sede", func(p *domain.NewSaleParams) { p.LocationID = uuid.Nil }, domain.ErrInvalidLocationID},
			{"sin usuario", func(p *domain.NewSaleParams) { p.UserID = uuid.Nil }, domain.ErrInvalidUserID},
			{"cliente vacío", func(p *domain.NewSaleParams) { p.CustomerID = &nilCustomer }, domain.ErrInvalidCustomerID},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := valid()
				tt.mutate(&p)
				_, err := domain.NewSale(p)
				require.ErrorIs(t, err, tt.want)
			})
		}
	})
}

func TestSaleTotals(t *testing.T) {
	t.Run("calcula totales con IVA de 0%, 5% y 19% agrupados por tasa", func(t *testing.T) {
		totals := mixedSale(t).Totals()

		requireDec(t, "30601.50", totals.Subtotal)
		requireDec(t, "1000", totals.Discounts)
		requireDec(t, "2109.29", totals.Tax)
		requireDec(t, "31710.79", totals.Total)

		require.Len(t, totals.Taxes, 3)
		wants := []struct{ rate, base, tax string }{
			{"0", "4500", "0"},
			{"0.05", "19000", "950"},
			{"0.19", "6101.50", "1159.29"},
		}
		for i, w := range wants {
			requireDec(t, w.rate, totals.Taxes[i].Rate)
			requireDec(t, w.base, totals.Taxes[i].Base)
			requireDec(t, w.tax, totals.Taxes[i].Tax)
		}
	})

	t.Run("calcula los montos de cada línea", func(t *testing.T) {
		s := newSale(t)
		in := line("2", "10000", "0.05", "1000")
		addLine(t, s, in)

		require.Len(t, s.Lines(), 1)
		l := s.Lines()[0]
		require.Equal(t, in.ProductID, l.ProductID())
		require.Equal(t, "Producto", l.ProductName())
		requireDec(t, "2", l.Quantity())
		requireDec(t, "10000", l.UnitPrice())
		requireDec(t, "0.05", l.TaxRate())
		requireDec(t, "1000", l.Discount())
		requireDec(t, "20000", l.Gross())
		requireDec(t, "19000", l.Base())
		requireDec(t, "950", l.Tax())
		requireDec(t, "19950", l.Total())
	})

	t.Run("redondea half-up a 2 decimales el bruto de una cantidad fraccionaria", func(t *testing.T) {
		tests := []struct{ qty, price, gross string }{
			{"2.5", "1234.567", "3086.42"},   // 3086.4175
			{"0.5", "1000.01", "500.01"},     // 500.005
			{"1.25", "1000.0001", "1250.00"}, // 1250.000125
		}
		for _, tt := range tests {
			s := newSale(t)
			addLine(t, s, line(tt.qty, tt.price, "0", ""))
			requireDec(t, tt.gross, firstLine(t, s).Gross())
		}
	})

	t.Run("el precio de góndola 8403.36 + IVA da 10000.00 por unidad", func(t *testing.T) {
		s := newSale(t)
		addLine(t, s, line("1", "8403.36", "0.19", ""))

		requireDec(t, "1596.64", firstLine(t, s).Tax())
		requireDec(t, "10000.00", s.Totals().Total)
	})

	t.Run("una venta sin líneas tiene totales en cero", func(t *testing.T) {
		totals := newSale(t).Totals()
		require.True(t, totals.Total.IsZero())
		require.Empty(t, totals.Taxes)
	})

	t.Run("modificar las líneas devueltas no altera la venta", func(t *testing.T) {
		s := mixedSale(t)
		lines := s.Lines()
		require.Len(t, lines, 4)
		lines[0] = lines[1]
		require.NotEqual(t, s.Lines()[0].ProductID(), s.Lines()[1].ProductID())
	})
}

func TestSaleAddLine(t *testing.T) {
	t.Run("rechaza datos inválidos de la línea", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(*domain.LineInput)
			want   error
		}{
			{"sin producto", func(in *domain.LineInput) { in.ProductID = uuid.Nil }, domain.ErrInvalidProductID},
			{"nombre vacío", func(in *domain.LineInput) { in.ProductName = "  " }, domain.ErrInvalidProductName},
			{"nombre de más de 200 caracteres", func(in *domain.LineInput) { in.ProductName = strings.Repeat("ñ", 201) }, domain.ErrInvalidProductName},
			{"cantidad cero", func(in *domain.LineInput) { in.Quantity = dec("0") }, domain.ErrInvalidQuantity},
			{"cantidad negativa", func(in *domain.LineInput) { in.Quantity = dec("-1") }, domain.ErrInvalidQuantity},
			{"cantidad con más de 4 decimales", func(in *domain.LineInput) { in.Quantity = dec("1.00001") }, domain.ErrInvalidQuantity},
			{"precio cero", func(in *domain.LineInput) { in.UnitPrice = dec("0") }, domain.ErrInvalidUnitPrice},
			{"precio con más de 4 decimales", func(in *domain.LineInput) { in.UnitPrice = dec("10.00001") }, domain.ErrInvalidUnitPrice},
			{"bruto que no cabe", func(in *domain.LineInput) { in.Quantity, in.UnitPrice = dec("10000000"), dec("10000000") }, domain.ErrInvalidQuantity},
			{"tasa negativa", func(in *domain.LineInput) { in.TaxRate = dec("-0.19") }, domain.ErrInvalidTaxRate},
			{"tasa del 100%", func(in *domain.LineInput) { in.TaxRate = dec("1") }, domain.ErrInvalidTaxRate},
			{"descuento negativo", func(in *domain.LineInput) { in.Discount = dec("-1") }, domain.ErrInvalidDiscount},
			{"descuento con más de 2 decimales", func(in *domain.LineInput) { in.Discount = dec("10.001") }, domain.ErrInvalidDiscount},
			{"descuento mayor al bruto", func(in *domain.LineInput) { in.Discount = dec("20000.01") }, domain.ErrInvalidDiscount},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := newSale(t)
				in := line("2", "10000", "0.19", "")
				tt.mutate(&in)
				require.ErrorIs(t, s.AddLine(in, domain.NoDiscountLimit), tt.want)
				require.Empty(t, s.Lines())
			})
		}
	})

	t.Run("recorta el nombre del producto", func(t *testing.T) {
		s := newSale(t)
		in := line("1", "100", "0", "")
		in.ProductName = "  Cemento gris 50kg "
		addLine(t, s, in)
		require.Equal(t, "Cemento gris 50kg", firstLine(t, s).ProductName())
	})

	t.Run("permite el mismo producto en dos líneas", func(t *testing.T) {
		s := newSale(t)
		first := line("1", "100", "0", "")
		second := first
		second.Discount = dec("10")
		addLine(t, s, first)
		addLine(t, s, second)
		require.Len(t, s.Lines(), 2)
	})
}

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
