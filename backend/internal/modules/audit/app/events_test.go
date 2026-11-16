package app_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
)

// record stores an event of action by actor at the env's current time.
func (e *env) record(t *testing.T, actor app.Actor, action domain.Action, details map[string]string) {
	t.Helper()
	entity := newID()
	entry := app.Entry{Action: action, Details: details}
	if action.EntityType() != "module" {
		entry.EntityID = &entity
	}
	require.NoError(t, e.svc.Record(t.Context(), actor, entry))
}

func TestListEvents(t *testing.T) {
	price := map[string]string{"old_price": "4500.00", "new_price": "4800.00"}
	voided := map[string]string{"total": "52000.00"}

	t.Run("lista del más reciente al más antiguo, con el total", func(t *testing.T) {
		e := newEnv(t)
		e.record(t, e.actor, domain.ActionVariantPriceChanged, price)
		e.now = e.now.Add(time.Minute)
		e.record(t, e.actor, domain.ActionSaleVoided, voided)

		page, err := e.svc.ListEvents(t.Context(), e.actor, app.EventQuery{Limit: 1})
		require.NoError(t, err)
		require.Equal(t, 2, page.Total)
		require.Len(t, page.Items, 1)
		require.Equal(t, domain.ActionSaleVoided, page.Items[0].Action())
	})

	t.Run("filtra por acción y por usuario", func(t *testing.T) {
		e := newEnv(t)
		other := app.Actor{TenantID: e.actor.TenantID, UserID: newID(), Role: "cashier"}
		e.record(t, e.actor, domain.ActionVariantPriceChanged, price)
		e.record(t, other, domain.ActionSaleVoided, voided)
		e.record(t, e.actor, domain.ActionSaleVoided, voided)

		page, err := e.svc.ListEvents(t.Context(), e.actor, app.EventQuery{
			Action: "sale.voided", UserID: &other.UserID, Limit: 20,
		})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total)
		require.Equal(t, other.UserID, page.Items[0].UserID())
	})

	t.Run("filtra por días en Colombia, con los dos extremos incluidos", func(t *testing.T) {
		e := newEnv(t)
		// 2026-11-15 04:00 UTC is still the 14th in Colombia.
		e.now = time.Date(2026, 11, 15, 4, 0, 0, 0, time.UTC)
		e.record(t, e.actor, domain.ActionSaleVoided, voided)
		e.now = time.Date(2026, 11, 16, 4, 59, 0, 0, time.UTC)
		e.record(t, e.actor, domain.ActionSaleVoided, voided)
		e.now = time.Date(2026, 11, 16, 5, 0, 0, 0, time.UTC)
		e.record(t, e.actor, domain.ActionSaleVoided, voided)

		day := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
		page, err := e.svc.ListEvents(t.Context(), e.actor, app.EventQuery{From: &day, To: &day, Limit: 20})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total)
	})

	t.Run("no ve los eventos de otra empresa", func(t *testing.T) {
		e := newEnv(t)
		e.record(t, app.Actor{TenantID: newID(), UserID: newID(), Role: "owner"}, domain.ActionSaleVoided, voided)

		page, err := e.svc.ListEvents(t.Context(), e.actor, app.EventQuery{Limit: 20})
		require.NoError(t, err)
		require.Zero(t, page.Total)
	})

	t.Run("rechaza una acción fuera del catálogo", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.ListEvents(t.Context(), e.actor, app.EventQuery{Action: "sale.deleted", Limit: 20})
		require.ErrorIs(t, err, domain.ErrUnknownAction)
	})

	t.Run("rechaza un rango de fechas invertido", func(t *testing.T) {
		e := newEnv(t)
		from, to := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
		_, err := e.svc.ListEvents(t.Context(), e.actor, app.EventQuery{From: &from, To: &to, Limit: 20})
		require.ErrorIs(t, err, domain.ErrInvalidFilter)
	})

	t.Run("abre su span con el tenant", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.ListEvents(t.Context(), e.actor, app.EventQuery{UserID: new(uuid.UUID), Limit: 20})
		require.NoError(t, err)
		require.Equal(t, e.actor.TenantID.String(), e.spanAttrs(t, "audit.ListEvents")["tenant.id"].AsString())
	})
}
