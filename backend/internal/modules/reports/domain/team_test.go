package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

func TestSumByUser(t *testing.T) {
	ana, luis := uuid.MustParse("01920000-0000-7000-8000-000000000001"), uuid.MustParse("01920000-0000-7000-8000-000000000002")
	points := []domain.TeamPoint{
		{UserID: luis, Name: "Luis", BucketStart: day(time.November, 2), TeamFigures: domain.TeamFigures{
			Voids: 1, VoidsTotal: dec("35700"), CashDifferences: 1, CashShortage: dec("5000")}},
		{UserID: ana, Name: "Ana", BucketStart: day(time.November, 2), TeamFigures: domain.TeamFigures{
			Discounts: 2, DiscountsAmount: dec("10000"), DiscountsGross: dec("100000")}},
		{UserID: luis, Name: "Luis", BucketStart: day(time.November, 9), TeamFigures: domain.TeamFigures{
			Voids: 2, VoidsTotal: dec("64300"), Adjustments: 1, AdjustmentsValue: dec("-25000")}},
	}

	users := domain.SumByUser(points)

	require.Len(t, users, 2)
	require.Equal(t, "Ana", users[0].Name, "ordena por nombre")
	luisTotal := users[1]
	require.Equal(t, 3, luisTotal.Voids)
	require.True(t, dec("100000").Equal(luisTotal.VoidsTotal))
	require.True(t, dec("-25000").Equal(luisTotal.AdjustmentsValue))
	require.True(t, dec("5000").Equal(luisTotal.CashShortage))
	rate, ok := users[0].DiscountRate()
	require.True(t, ok)
	require.True(t, dec("0.1").Equal(rate))
	_, ok = luisTotal.DiscountRate()
	require.False(t, ok, "sin descuentos no hay porcentaje promedio")
}

func TestHideCosts(t *testing.T) {
	t.Run("el control del equipo oculta el valor de los ajustes", func(t *testing.T) {
		r := domain.TeamReport{
			Users:  []domain.TeamUser{{TeamFigures: domain.TeamFigures{Adjustments: 1, AdjustmentsValue: dec("-9000")}}},
			Series: []domain.TeamPoint{{TeamFigures: domain.TeamFigures{Adjustments: 1, AdjustmentsValue: dec("-9000")}}},
			Costs:  true,
		}
		r.HideCosts()
		require.False(t, r.Costs)
		require.True(t, r.Users[0].AdjustmentsValue.IsZero())
		require.True(t, r.Series[0].AdjustmentsValue.IsZero())
		require.Equal(t, 1, r.Users[0].Adjustments, "la cantidad de ajustes sigue")
	})
	t.Run("los productos vendidos ocultan el costo de cada producto y variante", func(t *testing.T) {
		r := domain.ProductsReport{
			Rows: []domain.ProductSales{{Profit: domain.Profit{Revenue: dec("50000"), Cost: dec("30000")},
				Variants: []domain.VariantSales{{Profit: domain.Profit{Revenue: dec("50000"), Cost: dec("30000")}}}}},
			Totals: domain.Profit{Revenue: dec("50000"), Cost: dec("30000")}, Costs: true,
		}
		r.HideCosts()
		require.False(t, r.Costs)
		require.True(t, r.Rows[0].Cost.IsZero())
		require.True(t, r.Rows[0].Variants[0].Cost.IsZero())
		require.True(t, r.Totals.Cost.IsZero())
		require.True(t, dec("50000").Equal(r.Rows[0].Revenue), "el ingreso sigue")
	})
	t.Run("la plata quieta y los ajustes ocultan su valor", func(t *testing.T) {
		idle := domain.IdleStockReport{Rows: []domain.IdleProduct{{Value: dec("1"),
			Variants: []domain.IdleVariant{{Value: dec("1")}}}}, Value: dec("1"), Costs: true}
		idle.HideCosts()
		require.True(t, idle.Value.IsZero())
		require.True(t, idle.Rows[0].Value.IsZero())
		require.True(t, idle.Rows[0].Variants[0].Value.IsZero())
		adj := domain.AdjustmentsReport{Rows: []domain.AdjustmentRow{{Value: dec("-1")}}, Value: dec("-1"), Costs: true}
		adj.HideCosts()
		require.True(t, adj.Value.IsZero())
		require.True(t, adj.Rows[0].Value.IsZero())
	})
}

func TestSums(t *testing.T) {
	t.Run("los totales de los cierres suman los grupos", func(t *testing.T) {
		r := domain.CashReport{Rows: []domain.CashRow{
			{CashFigures: domain.CashFigures{Closings: 2, WithDifference: 1, SalesTotal: dec("100"), Shortage: dec("5"), Difference: dec("-5")}},
			{CashFigures: domain.CashFigures{Closings: 1, WithDifference: 1, SalesTotal: dec("50"), Overage: dec("2"), Difference: dec("2")}},
		}}
		r.Sum()
		require.Equal(t, 3, r.Totals.Closings)
		require.Equal(t, 2, r.Totals.WithDifference)
		require.True(t, dec("150").Equal(r.Totals.SalesTotal))
		require.True(t, dec("-3").Equal(r.Totals.Difference))
		require.True(t, dec("5").Equal(r.Totals.Shortage))
		require.True(t, dec("2").Equal(r.Totals.Overage))
	})
	t.Run("los totales de los ajustes suman los grupos", func(t *testing.T) {
		r := domain.AdjustmentsReport{Rows: []domain.AdjustmentRow{
			{Adjustments: 2, Value: decimal.RequireFromString("-30000")},
			{Adjustments: 1, Value: decimal.RequireFromString("4500")},
		}}
		r.Sum()
		require.Equal(t, 3, r.Adjustments)
		require.True(t, dec("-25500").Equal(r.Value))
	})
}

func TestNewCashReport(t *testing.T) {
	ana, luis := uuid.MustParse("01920000-0000-7000-8000-000000000001"), uuid.MustParse("01920000-0000-7000-8000-000000000002")
	centro, norte := uuid.MustParse("01920000-0000-7000-8000-0000000000c1"), uuid.MustParse("01920000-0000-7000-8000-0000000000c2")
	closings := []domain.Closing{
		{UserID: ana, UserName: "Ana", LocationID: centro, LocationName: "Ferretería centro", Day: day(time.November, 3),
			SalesTotal: dec("500000"), Difference: dec("0")},
		{UserID: luis, UserName: "Luis", LocationID: norte, LocationName: "Almacén norte", Day: day(time.November, 3),
			SalesTotal: dec("300000"), Difference: dec("-8000")},
		{UserID: luis, UserName: "Luis", LocationID: norte, LocationName: "Almacén norte", Day: day(time.November, 4),
			SalesTotal: dec("200000"), Difference: dec("3000")},
	}

	t.Run("por cajero, el de más faltante primero, con faltantes y sobrantes por separado", func(t *testing.T) {
		r := domain.NewCashReport(closings, domain.CashByUser)
		require.Len(t, r.Rows, 2)
		l := r.Rows[0]
		require.Equal(t, "Luis", l.Label)
		require.Equal(t, 2, l.Closings)
		require.Equal(t, 2, l.WithDifference)
		require.True(t, dec("8000").Equal(l.Shortage))
		require.True(t, dec("3000").Equal(l.Overage))
		require.True(t, dec("-5000").Equal(l.Difference))
		require.Equal(t, 0, r.Rows[1].WithDifference)
	})
	t.Run("por día, en orden de fecha, y los totales suman todos los cierres", func(t *testing.T) {
		r := domain.NewCashReport(closings, domain.CashByDay)
		require.Equal(t, "2026-11-03", r.Rows[0].Key)
		require.Equal(t, 2, r.Rows[0].Closings)
		require.Equal(t, 3, r.Totals.Closings)
		require.True(t, dec("1000000").Equal(r.Totals.SalesTotal))
	})
	t.Run("por sede", func(t *testing.T) {
		r := domain.NewCashReport(closings, domain.CashByLocation)
		require.Equal(t, "Almacén norte", r.Rows[0].Label)
	})
}
