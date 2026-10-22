package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// confirmedSale is mixedSale paid in cash with 40000 (change 8289.21).
func confirmedSale(t *testing.T) *domain.Sale {
	t.Helper()
	s := mixedSale(t)
	require.NoError(t, s.Confirm([]domain.Payment{payment(t, domain.PaymentCash, "40000")}, testNow))
	return s
}

func TestSaleAssignNumber(t *testing.T) {
	t.Run("asigna el consecutivo una sola vez a una venta confirmada", func(t *testing.T) {
		s := confirmedSale(t)
		require.Zero(t, s.Number())

		require.NoError(t, s.AssignNumber(41))
		require.Equal(t, int64(41), s.Number())

		require.ErrorIs(t, s.AssignNumber(42), domain.ErrSaleNumberAssigned)
		require.Equal(t, int64(41), s.Number())
	})

	t.Run("rechaza un consecutivo cero o negativo", func(t *testing.T) {
		s := confirmedSale(t)
		require.ErrorIs(t, s.AssignNumber(0), domain.ErrInvalidSaleNumber)
		require.ErrorIs(t, s.AssignNumber(-1), domain.ErrInvalidSaleNumber)
		require.Zero(t, s.Number())
	})

	t.Run("rechaza asignar consecutivo a un borrador", func(t *testing.T) {
		require.ErrorIs(t, mixedSale(t).AssignNumber(1), domain.ErrSaleNotConfirmed)
	})
}

func TestSaleVoid(t *testing.T) {
	voidedAt := testNow.Add(2 * time.Hour)

	t.Run("anula una venta confirmada con motivo y fecha", func(t *testing.T) {
		s := confirmedSale(t)
		by := newID()

		require.NoError(t, s.Void(by, "  El cliente desistió  ", voidedAt))

		require.Equal(t, domain.SaleVoided, s.Status())
		v := s.VoidInfo()
		require.NotNil(t, v)
		require.Equal(t, by, v.UserID)
		require.Equal(t, "El cliente desistió", v.Reason)
		require.Equal(t, voidedAt.UTC(), v.At)
		requireDec(t, "31710.79", s.Totals().Total)
	})

	t.Run("una venta confirmada no tiene anulación", func(t *testing.T) {
		require.Nil(t, confirmedSale(t).VoidInfo())
	})

	t.Run("rechaza anular un borrador", func(t *testing.T) {
		s := mixedSale(t)
		require.ErrorIs(t, s.Void(newID(), "Error de digitación", voidedAt), domain.ErrSaleNotConfirmed)
		require.Equal(t, domain.SaleDraft, s.Status())
	})

	t.Run("rechaza anular una venta ya anulada", func(t *testing.T) {
		s := confirmedSale(t)
		first := newID()
		require.NoError(t, s.Void(first, "Error de digitación", voidedAt))

		require.ErrorIs(t, s.Void(newID(), "Otra vez", voidedAt), domain.ErrSaleNotConfirmed)
		require.Equal(t, first, s.VoidInfo().UserID)
	})

	t.Run("rechaza anular sin motivo o con motivo de más de 500 caracteres", func(t *testing.T) {
		for _, reason := range []string{"", "   ", strings.Repeat("a", 501)} {
			s := confirmedSale(t)
			require.ErrorIs(t, s.Void(newID(), reason, voidedAt), domain.ErrInvalidVoidReason)
			require.Equal(t, domain.SaleConfirmed, s.Status())
		}
		require.NoError(t, confirmedSale(t).Void(newID(), strings.Repeat("ñ", 500), voidedAt))
	})

	t.Run("rechaza anular sin usuario", func(t *testing.T) {
		s := confirmedSale(t)
		require.ErrorIs(t, s.Void([16]byte{}, "Error", voidedAt), domain.ErrInvalidUserID)
	})
}

func TestInsufficientStockError(t *testing.T) {
	t.Run("es ErrInsufficientStock y nombra la variante", func(t *testing.T) {
		id := newID()
		var err error = &domain.InsufficientStockError{VariantID: id, ProductName: "Cemento gris 50kg"}

		require.ErrorIs(t, err, domain.ErrInsufficientStock)
		var typed *domain.InsufficientStockError
		require.True(t, errors.As(err, &typed))
		require.Equal(t, id, typed.VariantID)
		require.Contains(t, err.Error(), id.String())
		require.NotContains(t, err.Error(), "Cemento", "the message goes to logs and spans")
	})

	t.Run("describe el producto con la etiqueta de la variante", func(t *testing.T) {
		simple := &domain.InsufficientStockError{VariantID: newID(), ProductName: "Cemento gris 50kg"}
		variant := &domain.InsufficientStockError{VariantID: newID(), ProductName: "Camiseta básica", VariantLabel: "Rojo · M"}
		require.Equal(t, "Cemento gris 50kg", simple.Description())
		require.Equal(t, "Camiseta básica (Rojo · M)", variant.Description())
	})
}

// snapshotOf copies a confirmed sale into a snapshot, as the repository
// would store it.
func snapshotOf(s *domain.Sale) domain.SaleSnapshot {
	snap := domain.SaleSnapshot{
		ID: s.ID(), TenantID: s.TenantID(), LocationID: s.LocationID(), UserID: s.UserID(),
		CustomerID: s.CustomerID(), Number: s.Number(), Status: s.Status(),
		CreatedAt: s.CreatedAt(), ConfirmedAt: *s.ConfirmedAt(), Void: s.VoidInfo(),
	}
	for _, l := range s.Lines() {
		snap.Lines = append(snap.Lines, domain.LineSnapshot{
			ProductID: l.ProductID(), VariantID: l.VariantID(), ProductName: l.ProductName(),
			VariantLabel: l.VariantLabel(), Quantity: l.Quantity(), UnitPrice: l.UnitPrice(), TaxRate: l.TaxRate(),
			Discount: l.Discount(), Gross: l.Gross(), Tax: l.Tax(), PriceRule: l.PriceRule(), Unit: l.Unit(),
		})
	}
	for _, p := range s.Payments() {
		snap.Payments = append(snap.Payments, domain.PaymentSnapshot{Method: p.Method(), Amount: p.Amount()})
	}
	t := s.Totals()
	snap.Totals = domain.StoredTotals{
		Subtotal: t.Subtotal, Discounts: t.Discounts, Tax: t.Tax, Total: t.Total,
		Paid: s.Paid(), Change: s.Change(),
	}
	return snap
}

func TestRehydrateSale(t *testing.T) {
	numbered := func(t *testing.T) *domain.Sale {
		s := confirmedSale(t)
		require.NoError(t, s.AssignNumber(7))
		return s
	}

	t.Run("reconstruye una venta guardada con sus líneas, pagos y totales", func(t *testing.T) {
		orig := numbered(t)

		got, err := domain.RehydrateSale(snapshotOf(orig))
		require.NoError(t, err)

		require.Equal(t, orig.ID(), got.ID())
		require.Equal(t, int64(7), got.Number())
		require.Equal(t, domain.SaleConfirmed, got.Status())
		require.Len(t, got.Lines(), 4)
		require.Equal(t, orig.Lines(), got.Lines())
		require.Equal(t, orig.Totals(), got.Totals())
		requireDec(t, "8289.21", got.Change())
		require.Equal(t, orig.ConfirmedAt(), got.ConfirmedAt())
		require.Nil(t, got.VoidInfo())
	})

	t.Run("conserva la variante, la etiqueta y la regla de precio de cada línea", func(t *testing.T) {
		orig := newSale(t)
		in := line("2", "25000", "0.19", "")
		in.ProductName, in.VariantLabel, in.PriceRule = "Camiseta básica", "Rojo · M", "price_lists"
		addLine(t, orig, in)
		addLine(t, orig, line("2.5", "1234.567", "0.19", ""))
		require.NoError(t, orig.Confirm([]domain.Payment{payment(t, domain.PaymentCash, "100000")}, testNow))
		require.NoError(t, orig.AssignNumber(1))

		got, err := domain.RehydrateSale(snapshotOf(orig))
		require.NoError(t, err)
		require.Equal(t, orig.Lines(), got.Lines())
		require.Equal(t, "Rojo · M", got.Lines()[0].VariantLabel())
		require.Equal(t, domain.PriceRuleID("price_lists"), got.Lines()[0].PriceRule())
	})

	t.Run("reconstruye una venta anulada", func(t *testing.T) {
		orig := numbered(t)
		require.NoError(t, orig.Void(newID(), "Error de digitación", testNow))

		got, err := domain.RehydrateSale(snapshotOf(orig))
		require.NoError(t, err)
		require.Equal(t, domain.SaleVoided, got.Status())
		require.Equal(t, orig.VoidInfo(), got.VoidInfo())
	})

	t.Run("rechaza reconstruir una venta cuyos totales guardados no cuadran con sus líneas", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(*domain.SaleSnapshot)
		}{
			{"IVA de una línea", func(s *domain.SaleSnapshot) { s.Lines[0].Tax = s.Lines[0].Tax.Add(dec("0.01")) }},
			{"bruto de una línea", func(s *domain.SaleSnapshot) { s.Lines[1].Gross = dec("1") }},
			{"total", func(s *domain.SaleSnapshot) { s.Totals.Total = s.Totals.Total.Add(dec("1")) }},
			{"subtotal", func(s *domain.SaleSnapshot) { s.Totals.Subtotal = dec("0") }},
			{"pagado", func(s *domain.SaleSnapshot) { s.Totals.Paid = dec("50000") }},
			{"cambio", func(s *domain.SaleSnapshot) { s.Totals.Change = dec("0") }},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				snap := snapshotOf(numbered(t))
				tt.mutate(&snap)
				_, err := domain.RehydrateSale(snap)
				require.ErrorIs(t, err, domain.ErrInconsistentSale)
			})
		}
	})

	t.Run("rechaza estados, consecutivos y anulaciones inválidos", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(*domain.SaleSnapshot)
		}{
			{"borrador", func(s *domain.SaleSnapshot) { s.Status = domain.SaleDraft }},
			{"estado desconocido", func(s *domain.SaleSnapshot) { s.Status = "pending" }},
			{"sin consecutivo", func(s *domain.SaleSnapshot) { s.Number = 0 }},
			{"sin líneas", func(s *domain.SaleSnapshot) { s.Lines = nil }},
			{"anulada sin datos de anulación", func(s *domain.SaleSnapshot) { s.Status = domain.SaleVoided }},
			{"confirmada con datos de anulación", func(s *domain.SaleSnapshot) {
				s.Void = &domain.VoidInfo{UserID: newID(), Reason: "x", At: testNow}
			}},
			{"pago con medio desconocido", func(s *domain.SaleSnapshot) { s.Payments[0].Method = "bitcoin" }},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				snap := snapshotOf(numbered(t))
				tt.mutate(&snap)
				_, err := domain.RehydrateSale(snap)
				require.Error(t, err)
			})
		}
	})
}
