package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

var (
	cement, shirt             = uuid.MustParse("01920000-0000-7000-8000-000000000a01"), uuid.MustParse("01920000-0000-7000-8000-000000000a02")
	cementBag, shirtM, shirtL = uuid.MustParse("01920000-0000-7000-8000-000000000b01"), uuid.MustParse("01920000-0000-7000-8000-000000000b02"), uuid.MustParse("01920000-0000-7000-8000-000000000b03")
)

// soldLines are a hardware store's cement and a clothing store's shirt in
// two sizes: the shirt sells less but earns more.
func soldLines() []domain.SoldVariant {
	return []domain.SoldVariant{
		{ProductID: cement, ProductName: "Cemento gris 50kg", UnitCode: "und", VariantID: cementBag, SKU: "CEM-50",
			Quantity: dec("10"), Profit: domain.Profit{Revenue: dec("300000"), Cost: dec("250000")}},
		{ProductID: shirt, ProductName: "Camiseta básica", UnitCode: "und", VariantID: shirtM, Label: "Rojo · M", SKU: "CAM-M",
			Quantity: dec("3"), Profit: domain.Profit{Revenue: dec("150000"), Cost: dec("60000")}},
		{ProductID: shirt, ProductName: "Camiseta básica", UnitCode: "und", VariantID: shirtL, Label: "Rojo · L", SKU: "CAM-L",
			Quantity: dec("4"), Profit: domain.Profit{Revenue: dec("200000"), Cost: dec("80000")}},
	}
}

func TestNewProductsReport(t *testing.T) {
	tests := []struct {
		name  string
		sort  domain.ProductSort
		first uuid.UUID
	}{
		{"por ingreso, la camiseta (350.000) antes que el cemento", domain.ByRevenue, shirt},
		{"por cantidad, el cemento (10) antes que la camiseta (7)", domain.ByQuantity, cement},
		{"por utilidad, la camiseta (210.000) antes que el cemento (50.000)", domain.ByProfit, shirt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := domain.NewProductsReport(soldLines(), tt.sort, domain.Page{Limit: 20})
			require.Equal(t, tt.first, r.Rows[0].ProductID)
		})
	}

	t.Run("agrupa por producto con el detalle por variante, la que más vendió primero", func(t *testing.T) {
		r := domain.NewProductsReport(soldLines(), domain.ByRevenue, domain.Page{Limit: 20})
		require.Equal(t, 2, r.Total)
		s := r.Rows[0]
		require.Equal(t, "Camiseta básica", s.Name)
		require.True(t, dec("7").Equal(s.Quantity))
		require.True(t, dec("350000").Equal(s.Revenue))
		require.True(t, dec("140000").Equal(s.Cost))
		require.Len(t, s.Variants, 2)
		require.Equal(t, "Rojo · L", s.Variants[0].Label)
		require.True(t, dec("650000").Equal(r.Totals.Revenue))
		require.True(t, dec("390000").Equal(r.Totals.Cost))
	})
	t.Run("pagina por producto y suma los totales de todos", func(t *testing.T) {
		r := domain.NewProductsReport(soldLines(), domain.ByRevenue, domain.Page{Limit: 1, Offset: 1})
		require.Len(t, r.Rows, 1)
		require.Equal(t, cement, r.Rows[0].ProductID)
		require.Equal(t, 2, r.Total)
		require.True(t, dec("650000").Equal(r.Totals.Revenue))
		empty := domain.NewProductsReport(soldLines(), domain.ByRevenue, domain.Page{Limit: 20, Offset: 5})
		require.Empty(t, empty.Rows)
	})
}

func TestNewIdleStockReport(t *testing.T) {
	sold := time.Date(2026, 8, 1, 15, 0, 0, 0, time.UTC)
	older := time.Date(2026, 6, 1, 15, 0, 0, 0, time.UTC)
	lines := []domain.IdleLine{
		{ProductID: cement, ProductName: "Cemento gris 50kg", UnitCode: "und", VariantID: cementBag, SKU: "CEM-50",
			Quantity: dec("40"), Value: dec("1000000"), LastSoldAt: &sold},
		{ProductID: shirt, ProductName: "Camiseta básica", UnitCode: "und", VariantID: shirtM, Label: "Rojo · M", SKU: "CAM-M",
			Quantity: dec("30"), Value: dec("600000"), LastSoldAt: &older},
		{ProductID: shirt, ProductName: "Camiseta básica", UnitCode: "und", VariantID: shirtL, Label: "Rojo · L", SKU: "CAM-L",
			Quantity: dec("25"), Value: dec("500000")},
	}

	t.Run("por valor, la camiseta (1.100.000) antes que el cemento; la última venta es la más reciente", func(t *testing.T) {
		r := domain.NewIdleStockReport(lines, true, domain.Page{Limit: 20})
		require.Equal(t, shirt, r.Rows[0].ProductID)
		require.True(t, dec("1100000").Equal(r.Rows[0].Value))
		require.True(t, dec("55").Equal(r.Rows[0].Quantity))
		require.Equal(t, &older, r.Rows[0].LastSoldAt)
		require.Nil(t, r.Rows[0].Variants[1].LastSoldAt, "la talla L nunca se vendió")
		require.True(t, dec("2100000").Equal(r.Value))
		require.Equal(t, 2, r.Total)
	})
	t.Run("sin ver costos, por cantidad", func(t *testing.T) {
		r := domain.NewIdleStockReport(lines, false, domain.Page{Limit: 1})
		require.Equal(t, shirt, r.Rows[0].ProductID)
		require.Len(t, r.Rows, 1)
	})
}
