package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
)

func TestCashClosings(t *testing.T) {
	e := newEnv(t)
	centro, norte := newID(), newID()
	e.readers.closings = []domain.Closing{
		{LocationID: centro, LocationName: "Centro", SalesTotal: dec("180000"), Difference: dec("-5000")},
		{LocationID: centro, LocationName: "Centro", SalesTotal: dec("120000")},
		{LocationID: norte, LocationName: "Norte", SalesTotal: dec("120000")},
	}

	r, err := e.svc.CashClosings(t.Context(), e.actor, app.CashQuery{Range: nov(1, 19), Group: "location"})

	require.NoError(t, err)
	require.Equal(t, domain.CashByLocation, r.Group)
	require.Len(t, r.Rows, 2)
	require.Equal(t, 3, r.Totals.Closings)
	require.True(t, dec("420000").Equal(r.Totals.SalesTotal))
	require.True(t, dec("-5000").Equal(r.Totals.Difference))
}

func TestTeamControl(t *testing.T) {
	ana, luis := newID(), newID()
	week := calendar.NewDate(2026, time.November, 2)
	points := []domain.TeamPoint{
		{UserID: luis, Name: "Luis", BucketStart: week, TeamFigures: domain.TeamFigures{Voids: 2, VoidsTotal: dec("71400"), Adjustments: 1, AdjustmentsValue: dec("-25000")}},
		{UserID: ana, Name: "Ana", BucketStart: week, TeamFigures: domain.TeamFigures{Discounts: 1, DiscountsAmount: dec("5000"), DiscountsGross: dec("50000")}},
	}

	t.Run("suma por usuario y devuelve la tendencia por semana", func(t *testing.T) {
		e := newEnv(t)
		e.readers.points = points
		r, err := e.svc.TeamControl(t.Context(), e.actor, app.TeamQuery{Range: nov(1, 19)})
		require.NoError(t, err)
		require.Equal(t, domain.BucketWeek, e.readers.teamFilter.Bucket)
		require.Len(t, r.Users, 2)
		require.Equal(t, "Ana", r.Users[0].Name)
		require.Equal(t, 2, r.Users[1].Voids)
		require.Len(t, r.Series, 2)
		require.True(t, dec("-25000").Equal(r.Users[1].AdjustmentsValue))
	})
	t.Run("oculta el valor de los ajustes a quien no ve costos", func(t *testing.T) {
		e := newEnv(t)
		e.readers.points = append([]domain.TeamPoint(nil), points...)
		r, err := e.svc.TeamControl(t.Context(), e.cashier(), app.TeamQuery{Range: nov(1, 19), Bucket: "day"})
		require.NoError(t, err)
		require.False(t, r.Costs)
		require.True(t, r.Users[1].AdjustmentsValue.IsZero())
		require.Equal(t, domain.BucketDay, r.Bucket)
	})
	t.Run("rechaza un periodo de más de un año", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.TeamControl(t.Context(), e.actor, app.TeamQuery{Range: app.Range{
			From: calendar.NewDate(2025, time.January, 1), To: calendar.NewDate(2026, time.November, 19)}})
		require.ErrorIs(t, err, domain.ErrInvalidPeriod)
	})
}

func TestReceivablesAging(t *testing.T) {
	e := newEnv(t)
	e.aging.report = domain.AgingExport{Lines: []domain.AgingLine{{Name: "Constructora Andina", Total: dec("800000")}}}

	r, err := e.svc.ReceivablesAging(t.Context(), e.actor, "total")

	require.NoError(t, err)
	require.Equal(t, "total", e.aging.sort)
	require.Equal(t, e.actor.TenantID, e.aging.tenantID)
	require.Len(t, r.Lines, 1)
	e.span(t, "reports.ReceivablesAging")
}

func TestExported(t *testing.T) {
	e := newEnv(t)
	e.svc.Exported(t.Context(), "sales", "xlsx")
	require.EqualValues(t, 1, e.counter(t, "cuadra.reports.exports", "format", "xlsx"))
}
