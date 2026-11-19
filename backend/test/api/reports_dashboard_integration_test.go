//go:build integration

package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestReportsDashboardFigures checks what the owner's dashboard asks of the
// reports: the sales up to a time of day, and the idle stock after a void.
func TestReportsDashboardFigures(t *testing.T) {
	f := newReportsFlow(t)
	voided := f.sell(t, "1", "35700")
	rec := f.do(t, http.MethodPost, "/api/v1/sales/"+voided+"/void", `{"reason":"El cliente desistió"}`, f.ownerToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	t.Run("sin movimiento: un producto cuya única venta se anuló aparece como sin vender", func(t *testing.T) {
		var idle idleReport
		f.report(t, "/api/v1/reports/idle-stock?days=30&location_id="+f.locationID, f.ownerToken, &idle)
		require.Len(t, idle.Items, 1)
		require.Equal(t, "Cemento gris 50kg", idle.Items[0].Name)
		require.Nil(t, idle.Items[0].LastSoldAt)
	})

	f.sell(t, "2", "71400")

	t.Run("con hora de corte, las ventas posteriores a esa hora del último día no cuentan", func(t *testing.T) {
		var whole, cut salesReport
		f.report(t, "/api/v1/reports/sales?"+f.period, f.ownerToken, &whole)
		f.report(t, "/api/v1/reports/sales?until=00:00&"+f.period, f.ownerToken, &cut)
		require.Equal(t, 1, whole.Totals.SalesCount)
		require.Equal(t, "71400.00", whole.Totals.Total)
		require.Equal(t, 0, cut.Totals.SalesCount)
		require.Equal(t, "0.00", cut.Totals.Total)
	})
	t.Run("una hora de corte inválida responde 422 invalid_period", func(t *testing.T) {
		rec := f.do(t, http.MethodGet, "/api/v1/reports/sales?until=25:00&"+f.period, "", f.ownerToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "invalid_period")
	})
}
