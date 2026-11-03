package app_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestCloseCashSessionOfAnotherUser(t *testing.T) {
	t.Run("quien administra la caja cierra la que dejó abierta un cajero", func(t *testing.T) {
		e := newEnv(t)
		s := e.openCash(t, newID(), "50000")
		admin := app.Actor{TenantID: e.actor.TenantID, UserID: newID(), ManagesCash: true}

		view, err := e.svc.CloseCashSession(t.Context(), admin, s.ID(), app.CloseCashSessionInput{Counts: counts("50000", 1)})
		require.NoError(t, err)

		require.Equal(t, domain.CashSessionClosed, view.Session.Status())
		require.Equal(t, admin.UserID, view.Session.Closing().ClosedBy)
	})

	t.Run("otro cajero no la cierra", func(t *testing.T) {
		e := newEnv(t)
		s := e.openCash(t, newID(), "50000")
		other := app.Actor{TenantID: e.actor.TenantID, UserID: newID()}

		_, err := e.svc.CloseCashSession(t.Context(), other, s.ID(), app.CloseCashSessionInput{Counts: counts("50000", 1)})
		require.ErrorIs(t, err, domain.ErrCashSessionNotOwner)
	})
}

func TestListCashSessions(t *testing.T) {
	t.Run("lista las cajas con el nombre de su cajero y pide los días en hora de Colombia", func(t *testing.T) {
		e := newEnv(t)
		location := newID()
		s := e.openCash(t, location, "50000")
		e.users.names[e.actor.UserID] = "Laura Pérez"
		from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)

		page, err := e.svc.ListCashSessions(t.Context(), e.actor, app.CashSessionQuery{
			LocationID: &location, Status: "open", From: &from, To: &to, Limit: 20,
		})
		require.NoError(t, err)

		require.Equal(t, 1, page.Total)
		require.Equal(t, s.ID(), page.Items[0].ID)
		require.Equal(t, "Laura Pérez", page.Items[0].CashierName)
		f := e.store.cashFilter
		require.Equal(t, domain.CashSessionOpen, f.Status)
		require.Equal(t, time.Date(2026, 10, 1, 5, 0, 0, 0, time.UTC), *f.OpenedFrom)
		require.Equal(t, time.Date(2026, 11, 1, 5, 0, 0, 0, time.UTC), *f.OpenedBefore)
		e.span(t, "sales.ListCashSessions")
	})

	t.Run("un cajero que ya no existe queda sin nombre, y rechaza un estado desconocido", func(t *testing.T) {
		e := newEnv(t)
		e.openCash(t, newID(), "0")

		page, err := e.svc.ListCashSessions(t.Context(), e.actor, app.CashSessionQuery{Limit: 20})
		require.NoError(t, err)
		require.Empty(t, page.Items[0].CashierName)

		_, err = e.svc.ListCashSessions(t.Context(), e.actor, app.CashSessionQuery{Status: "lost", Limit: 20})
		require.ErrorIs(t, err, app.ErrInvalidCashStatus)
	})

	t.Run("no ve las cajas de otra empresa", func(t *testing.T) {
		e := newEnv(t)
		e.openCash(t, newID(), "0")
		page, err := e.svc.ListCashSessions(t.Context(), app.Actor{TenantID: newID(), UserID: uuid.Must(uuid.NewV7())}, app.CashSessionQuery{Limit: 20})
		require.NoError(t, err)
		require.Empty(t, page.Items)
	})
}
