//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestReportsSalesAndProfit sells cement at two average costs, voids a sale
// and closes the cash short, and checks that the reports agree with the
// sales, the cash closings and the audit log.
func TestReportsSalesAndProfit(t *testing.T) {
	f := newReportsFlow(t)

	// Two bags at the cost of 25.000; then 10 more arrive at 35.000 (the
	// average goes to 30.000) and one more bag sells. 30.000 + 19 % IVA =
	// 35.700 a bag.
	f.sell(t, "2", "71400")
	f.receive(t, f.variantID, "8", "35000")
	f.sell(t, "1", "35700")
	voided := f.sell(t, "1", "35700")
	rec := f.do(t, http.MethodPost, "/api/v1/sales/"+voided+"/void", `{"reason":"El cliente desistió"}`, f.ownerToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	// 100.000 + 71.400 + 35.700 = 207.100 expected; 207.000 counted.
	rec = f.do(t, http.MethodPost, "/api/v1/cash-sessions/"+f.sessionID+"/close",
		`{"counts":[{"denomination":"100000","quantity":2},{"denomination":"5000","quantity":1},{"denomination":"2000","quantity":1}]}`, f.cashierToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	t.Run("la utilidad usa el costo vigente al momento de la venta, no el actual", func(t *testing.T) {
		var r productsReport
		f.report(t, "/api/v1/reports/products?"+f.period, f.ownerToken, &r)
		require.Len(t, r.Items, 1)
		cement := r.Items[0]
		require.Equal(t, "3", cement.Quantity, "la venta anulada no cuenta")
		require.Equal(t, "90000.00", cement.Revenue)
		require.Equal(t, "80000.00", *cement.Cost, "2 × 25.000 + 1 × 30.000")
		require.Equal(t, "10000.00", *cement.Profit)
		require.Equal(t, "0.1111", *cement.Margin)
	})
	t.Run("los totales del reporte de ventas cuadran con los cierres de caja del periodo", func(t *testing.T) {
		var sales salesReport
		f.report(t, "/api/v1/reports/sales?"+f.period, f.ownerToken, &sales)
		require.Equal(t, 2, sales.Totals.SalesCount)
		require.Equal(t, "107100.00", sales.Totals.Total)
		require.Equal(t, "90000.00", sales.Totals.Net)
		var cash cashReport
		f.report(t, "/api/v1/reports/cash-closings?"+f.period, f.ownerToken, &cash)
		require.Equal(t, sales.Totals.Total, cash.Totals.SalesTotal)
		require.Equal(t, 1, cash.Totals.Closings)
	})
	t.Run("por medio de pago suma lo pagado sin el cambio", func(t *testing.T) {
		var sales salesReport
		f.report(t, "/api/v1/reports/sales?group_by=payment_method&"+f.period, f.ownerToken, &sales)
		require.Len(t, sales.Items, 1)
		require.Equal(t, "cash", sales.Items[0].Key)
		require.Equal(t, "107100.00", sales.Items[0].Total)
		require.Nil(t, sales.Items[0].Net)
	})
	t.Run("las anulaciones y diferencias del control cuadran con la auditoría y los cierres", func(t *testing.T) {
		var team teamReport
		f.report(t, "/api/v1/reports/team-control?bucket=day&"+f.period, f.ownerToken, &team)
		voids, shortage, differences := 0, "", 0
		for _, u := range team.Users {
			voids += u.Voids
			differences += u.CashDifferences
			if u.CashDifferences > 0 {
				shortage = u.CashShortage
			}
		}
		audit := f.auditTotal(t, "sale.voided") // eventos del periodo
		require.Equal(t, audit, voids)
		require.Equal(t, 1, voids)
		require.Equal(t, f.auditTotal(t, "cash.closed_with_difference"), differences)
		require.Equal(t, "100.00", shortage)

		var cash cashReport
		f.report(t, "/api/v1/reports/cash-closings?"+f.period, f.ownerToken, &cash)
		require.Equal(t, "100.00", cash.Totals.Shortage)
		require.Equal(t, "-100.00", cash.Totals.Difference)
		require.NotEmpty(t, team.Series)
	})
	t.Run("exporta el reporte de productos a Excel", func(t *testing.T) {
		rec := f.do(t, http.MethodGet, "/api/v1/reports/export?report=products&"+f.period, "", f.ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Header().Get("Content-Type"), "spreadsheetml")
		require.Positive(t, rec.Body.Len())
	})
}

// auditTotal is how many events of action the tenant's audit log has.
func (f *reportsFlow) auditTotal(t *testing.T, action string) int {
	t.Helper()
	rec := f.do(t, http.MethodGet, "/api/v1/audit-events?action="+action+"&"+f.period, "", f.ownerToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var page struct{ Total int }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	return page.Total
}
