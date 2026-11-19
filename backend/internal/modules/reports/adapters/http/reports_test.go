package httpadapter_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func nov3() domain.Period {
	p, _ := domain.NewPeriod(calendar.NewDate(2026, time.November, 3), calendar.NewDate(2026, time.November, 3))
	return p
}

func TestGetSalesReport(t *testing.T) {
	t.Run("traduce los filtros y responde el dinero como strings", func(t *testing.T) {
		location := uuid.Must(uuid.NewV7())
		uc := &stubUseCases{salesReport: domain.SalesReport{Period: nov3(), Group: domain.SalesByDay,
			Rows: []domain.SalesRow{{Key: "2026-11-03", Label: "2026-11-03", SalesFigures: domain.SalesFigures{
				Sales: 2, Net: dec("60000"), Tax: dec("11400"), Total: dec("71400"), HasBreakdown: true}}},
			Totals: domain.SalesFigures{Sales: 2, Net: dec("60000"), Tax: dec("11400"), Total: dec("71400")}}}

		rec := serve(t, uc, owner, "/api/v1/reports/sales?from=2026-11-03&to=2026-11-03&location_id="+location.String())

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"period":{"from":"2026-11-03","to":"2026-11-03"},"group_by":"day",
			"items":[{"key":"2026-11-03","label":"2026-11-03","sales_count":2,"net":"60000.00","tax":"11400.00","total":"71400.00"}],
			"totals":{"sales_count":2,"net":"60000.00","tax":"11400.00","total":"71400.00"}}`, rec.Body.String())
		require.Equal(t, owner.TenantID, uc.actor.TenantID)
		require.Equal(t, &location, uc.sales.LocationID)
		require.Equal(t, "2026-11-03", uc.sales.From.String())
	})
	t.Run("por medio de pago no trae neto ni IVA", func(t *testing.T) {
		uc := &stubUseCases{salesReport: domain.SalesReport{Period: nov3(), Group: domain.SalesByPaymentMethod,
			Rows: []domain.SalesRow{{Key: "cash", Label: "cash", SalesFigures: domain.SalesFigures{Sales: 1, Total: dec("35700")}}}}}
		rec := serve(t, uc, owner, "/api/v1/reports/sales?from=2026-11-03&to=2026-11-03&group_by=payment_method")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `{"key":"cash","label":"cash","sales_count":1,"total":"35700.00"}`)
	})
	t.Run("un periodo inválido responde 422 invalid_period", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrInvalidPeriod}
		requireError(t, serve(t, uc, owner, "/api/v1/reports/sales?from=2026-11-04&to=2026-11-03"), http.StatusUnprocessableEntity, "invalid_period")
	})
	t.Run("sin el periodo responde 400", func(t *testing.T) {
		requireError(t, serve(t, &stubUseCases{}, owner, "/api/v1/reports/sales"), http.StatusBadRequest, "invalid_request")
	})
}

func TestGetProductsReport(t *testing.T) {
	report := func(costs bool) domain.ProductsReport {
		p := domain.Profit{Revenue: dec("50000"), Cost: dec("30000")}
		return domain.ProductsReport{Period: nov3(), Total: 1, Totals: p, Costs: costs, Rows: []domain.ProductSales{{
			ProductID: uuid.Must(uuid.NewV7()), Name: "Camiseta básica", UnitCode: "und", Quantity: dec("2"), Profit: p,
			Variants: []domain.VariantSales{{VariantID: uuid.Must(uuid.NewV7()), Label: "Rojo · M", SKU: "CAM-M", Quantity: dec("2"), Profit: p}},
		}}}
	}
	t.Run("quien ve costos recibe costo, utilidad y margen", func(t *testing.T) {
		uc := &stubUseCases{productsReport: report(true)}
		rec := serve(t, uc, owner, "/api/v1/reports/products?from=2026-11-03&to=2026-11-03&attribute=material&attribute_value=algod%C3%B3n")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.True(t, uc.actor.SeesCosts)
		require.Contains(t, rec.Body.String(), `"cost":"30000.00","margin":"0.4000"`)
		require.Equal(t, "algodón", *uc.products.AttributeValue)
		require.Equal(t, 20, uc.products.Limit)
	})
	t.Run("a quien no ve costos no le manda costo, utilidad ni margen", func(t *testing.T) {
		uc := &stubUseCases{productsReport: report(false)}
		rec := serve(t, uc, admin, "/api/v1/reports/products?from=2026-11-03&to=2026-11-03")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.False(t, uc.actor.SeesCosts)
		for _, field := range []string{`"cost"`, `"profit"`, `"margin"`} {
			require.NotContains(t, rec.Body.String(), field)
		}
	})
	t.Run("ordenar por utilidad sin ver costos responde 403 costs_hidden", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrCostsHidden}
		requireError(t, serve(t, uc, admin, "/api/v1/reports/products?from=2026-11-03&to=2026-11-03&sort=profit"), http.StatusForbidden, "costs_hidden")
	})
}

func TestExportReport(t *testing.T) {
	t.Run("exporta las ventas a CSV con el nombre del reporte y su periodo", func(t *testing.T) {
		uc := &stubUseCases{salesReport: domain.SalesReport{Period: nov3(), Group: domain.SalesByPaymentMethod,
			Rows: []domain.SalesRow{{Key: "nequi", SalesFigures: domain.SalesFigures{Sales: 1, Total: dec("12500.5")}}}}}
		rec := serve(t, uc, owner, "/api/v1/reports/export?report=sales&format=csv&from=2026-11-03&to=2026-11-03&group_by=payment_method")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, "text/csv; charset=utf-8", rec.Header().Get("Content-Type"))
		require.Contains(t, rec.Header().Get("Content-Disposition"), `filename="reporte-sales-2026-11-03-a-2026-11-03.csv"`)
		require.Contains(t, rec.Body.String(), "Nequi;1;;;12500,5")
		require.Equal(t, []string{"sales.csv"}, uc.exported)
	})
	t.Run("un reporte con periodo sin las fechas responde 422 invalid_filter", func(t *testing.T) {
		rec := serve(t, &stubUseCases{}, owner, "/api/v1/reports/export?report=team_control")
		requireError(t, rec, http.StatusUnprocessableEntity, "invalid_filter")
	})
	t.Run("la cartera se exporta sin periodo, en xlsx por defecto", func(t *testing.T) {
		uc := &stubUseCases{aging: domain.AgingExport{Lines: []domain.AgingLine{{Name: "Constructora Andina", Total: dec("800000")}}}}
		rec := serve(t, uc, owner, "/api/v1/reports/export?report=receivables")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Header().Get("Content-Type"), "spreadsheetml")
		require.Equal(t, []string{"receivables.xlsx"}, uc.exported)
	})
}
