package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestSalesReport(t *testing.T) {
	t.Run("pide al lector el periodo en hora de Colombia, el tenant del token y el agrupamiento", func(t *testing.T) {
		e := newEnv(t)
		location := newID()

		r, err := e.svc.SalesReport(t.Context(), e.actor, app.SalesQuery{Range: nov(3, 4), LocationID: &location, Group: "payment_method"})

		require.NoError(t, err)
		f := e.readers.salesFilter
		require.Equal(t, e.actor.TenantID, f.TenantID)
		require.Equal(t, &location, f.LocationID)
		require.Equal(t, domain.SalesByPaymentMethod, f.Group)
		require.Equal(t, time.Date(2026, 11, 3, 5, 0, 0, 0, time.UTC), f.Period.Start().UTC())
		require.Equal(t, domain.SalesByPaymentMethod, r.Group)
		require.Equal(t, f.Period, r.Period)
	})
	t.Run("sin agrupamiento agrupa por día", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.SalesReport(t.Context(), e.actor, app.SalesQuery{Range: nov(3, 3)})
		require.NoError(t, err)
		require.Equal(t, domain.SalesByDay, e.readers.salesFilter.Group)
	})
	t.Run("con hora de corte, el último día del periodo termina a esa hora", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.SalesReport(t.Context(), e.actor, app.SalesQuery{Range: nov(12, 12), Until: "09:15"})
		require.NoError(t, err)
		require.Equal(t, time.Date(2026, 11, 12, 14, 15, 0, 0, time.UTC), e.readers.salesFilter.Period.End().UTC())
	})
	t.Run("rechaza una hora de corte inválida sin leer las ventas", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.SalesReport(t.Context(), e.actor, app.SalesQuery{Range: nov(12, 12), Until: "25:00"})
		require.ErrorIs(t, err, domain.ErrInvalidPeriod)
		require.Zero(t, e.readers.reads)
	})
	t.Run("rechaza un periodo invertido sin leer las ventas", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.SalesReport(t.Context(), e.actor, app.SalesQuery{Range: nov(4, 3)})
		require.ErrorIs(t, err, domain.ErrInvalidPeriod)
		require.Zero(t, e.readers.reads)
	})
	t.Run("abre el span reports.SalesReport y registra el error del lector", func(t *testing.T) {
		e := newEnv(t)
		e.readers.err = errors.New("db down")
		_, err := e.svc.SalesReport(t.Context(), e.actor, app.SalesQuery{Range: nov(3, 3)})
		require.Error(t, err)
		require.Equal(t, codes.Error, e.span(t, "reports.SalesReport").Status().Code)
	})
	t.Run("cuenta la consulta por reporte", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.SalesReport(t.Context(), e.actor, app.SalesQuery{Range: nov(3, 3)})
		require.NoError(t, err)
		require.EqualValues(t, 1, e.counter(t, "cuadra.reports.queries", "report", "sales"))
	})
}

func TestProductsReport(t *testing.T) {
	cement, shirt := newID(), newID()
	sold := []domain.SoldVariant{
		{ProductID: cement, ProductName: "Cemento gris", VariantID: newID(), Profit: domain.Profit{Revenue: dec("60000"), Cost: dec("50000")}},
		{ProductID: shirt, ProductName: "Camiseta básica", VariantID: newID(), Label: "Rojo · M", Profit: domain.Profit{Revenue: dec("50000"), Cost: dec("30000")}},
	}

	t.Run("quien ve costos recibe costo y utilidad", func(t *testing.T) {
		e := newEnv(t)
		e.readers.sold = sold
		r, err := e.svc.ProductsReport(t.Context(), e.actor, app.ProductsQuery{Range: nov(1, 19), Sort: "profit", Limit: 1})
		require.NoError(t, err)
		require.True(t, r.Costs)
		require.Len(t, r.Rows, 1, "pagina")
		require.Equal(t, shirt, r.Rows[0].ProductID, "la camiseta deja más utilidad")
		require.True(t, dec("30000").Equal(r.Rows[0].Variants[0].Cost))
		require.Equal(t, 2, r.Total)
	})
	t.Run("oculta costos a quien no ve costos", func(t *testing.T) {
		e := newEnv(t)
		e.readers.sold = sold
		r, err := e.svc.ProductsReport(t.Context(), e.cashier(), app.ProductsQuery{Range: nov(1, 19), Limit: 20})
		require.NoError(t, err)
		require.False(t, r.Costs)
		require.True(t, r.Rows[0].Cost.IsZero())
		require.True(t, r.Totals.Cost.IsZero())
		require.True(t, dec("60000").Equal(r.Rows[0].Revenue))
		require.Equal(t, cement, r.Rows[0].ProductID)
	})
	t.Run("rechaza ordenar por utilidad sin ver costos", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.ProductsReport(t.Context(), e.cashier(), app.ProductsQuery{Range: nov(1, 19), Sort: "profit", Limit: 20})
		require.ErrorIs(t, err, domain.ErrCostsHidden)
		require.Zero(t, e.readers.reads)
	})
	t.Run("filtra por categoría y por un atributo de la empresa", func(t *testing.T) {
		e := newEnv(t)
		category, key, value := newID(), "material", "algodón"
		_, err := e.svc.ProductsReport(t.Context(), e.actor, app.ProductsQuery{Range: nov(1, 19), Limit: 20,
			ProductFilters: app.ProductFilters{CategoryID: &category, Attribute: &key, AttributeValue: &value}})
		require.NoError(t, err)
		f := e.readers.productFilter
		require.Equal(t, &category, f.CategoryID)
		require.Equal(t, "material", f.Attribute)
		require.Equal(t, "algodón", f.AttributeValue)
	})
	t.Run("la utilidad por venta exige ver costos", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.SaleProfits(t.Context(), e.cashier(), app.SaleProfitsQuery{Range: nov(1, 19), Limit: 20})
		require.ErrorIs(t, err, domain.ErrCostsHidden)
		_, err = e.svc.SaleProfits(t.Context(), e.actor, app.SaleProfitsQuery{Range: nov(1, 19), Limit: 20})
		require.NoError(t, err)
		require.Equal(t, e.actor.TenantID, e.readers.saleProfitFilter.TenantID)
		e.span(t, "reports.SaleProfits")
	})
}
