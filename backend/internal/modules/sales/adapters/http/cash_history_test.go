package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func (s *stubUseCases) ListCashSessions(_ context.Context, a app.Actor, q app.CashSessionQuery) (app.CashSessionPage, error) {
	s.actor, s.cashQuery = a, q
	return s.cashPage, s.err
}

func TestListCashSessionsHandler(t *testing.T) {
	t.Run("pasa los filtros y responde cada caja con su cajero y su cierre", func(t *testing.T) {
		closed := anOpenSession(t).Session
		closer := newID()
		diff := dec("-4800")
		at := now.Add(8 * time.Hour)
		uc := &stubUseCases{cashPage: app.CashSessionPage{Total: 2, Items: []app.CashSessionRow{
			{CashSessionSummary: domain.CashSessionSummary{
				ID: closed.ID(), LocationID: closed.LocationID(), UserID: closed.UserID(), Status: domain.CashSessionClosed,
				OpeningFloat: dec("100000"), OpenedAt: now, ClosedAt: &at, ClosedBy: &closer, SalesCount: 12,
				SalesTotal: dec("845300"), Difference: &diff,
			}, CashierName: "Laura Pérez"},
			{CashSessionSummary: domain.CashSessionSummary{
				ID: newID(), LocationID: newID(), UserID: newID(), Status: domain.CashSessionOpen, OpeningFloat: dec("0"), OpenedAt: now,
			}},
		}}}
		location := newID()

		rec := serve(t, uc, &owner, http.MethodGet,
			"/api/v1/cash-sessions?location_id="+location.String()+"&status=closed&from=2026-10-01&to=2026-10-31&limit=10&offset=10", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		q := uc.cashQuery
		require.Equal(t, location, *q.LocationID)
		require.Equal(t, "closed", q.Status)
		require.Equal(t, "2026-10-01", q.From.Format(time.DateOnly))
		require.Equal(t, "2026-10-31", q.To.Format(time.DateOnly))
		require.Equal(t, 10, q.Limit)
		require.Equal(t, 10, q.Offset)

		var body struct {
			Items []map[string]any
			Total int
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, 2, body.Total)
		first, open := body.Items[0], body.Items[1]
		require.Equal(t, "Laura Pérez", first["cashier_name"])
		require.Equal(t, "shortage", first["result"])
		require.Equal(t, "-4800.00", first["difference"])
		require.Equal(t, "845300.00", first["sales_total"])
		require.Equal(t, closer.String(), first["closed_by"])
		require.Equal(t, "open", open["status"])
		require.NotContains(t, open, "result")
		require.NotContains(t, open, "cashier_name")
	})

	t.Run("todas no filtra por estado", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/cash-sessions?status=all", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Empty(t, uc.cashQuery.Status)
		require.Equal(t, 20, uc.cashQuery.Limit)
	})
}

func TestCloseCashSessionManages(t *testing.T) {
	t.Run("el dueño cierra como quien administra la caja; el cajero no", func(t *testing.T) {
		uc := &stubUseCases{view: aClosedSession(t)}
		body := `{"counts":[{"denomination":"50000","quantity":1}]}`
		id := newID().String()

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/cash-sessions/"+id+"/close", body)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.True(t, uc.actor.ManagesCash)
		require.Contains(t, rec.Body.String(), `"sales_total":`)

		p := cashier()
		rec = serve(t, uc, &p, http.MethodPost, "/api/v1/cash-sessions/"+id+"/close", body)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.False(t, uc.actor.ManagesCash)
	})
}
