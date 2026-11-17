package app_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
)

func TestRecord(t *testing.T) {
	t.Run("guarda el evento con quién, cuándo y dónde", func(t *testing.T) {
		e := newEnv(t)
		sale, location := newID(), newID()
		err := e.svc.Record(t.Context(), e.actor, app.Entry{
			Action: domain.ActionSaleVoided, EntityID: &sale, LocationID: &location,
			Details: map[string]string{"total": "185000.00"},
		})
		require.NoError(t, err)

		require.Len(t, e.events.stored, 1)
		got := e.events.stored[0]
		require.Equal(t, e.actor.TenantID, got.TenantID())
		require.Equal(t, e.actor.UserID, got.UserID())
		require.Equal(t, "admin", got.Role())
		require.Equal(t, "sale", got.EntityType())
		require.Equal(t, &sale, got.EntityID())
		require.Equal(t, &location, got.LocationID())
		require.Equal(t, testNow, got.OccurredAt())
		require.Equal(t, int64(1), e.counter(t, "cuadra.audit.events", "action", "sale.voided"))
		attrs := e.spanAttrs(t, "audit.Record")
		require.Equal(t, "sale.voided", attrs["audit.action"].AsString())
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
	})

	t.Run("rechaza una acción fuera del catálogo y no guarda nada", func(t *testing.T) {
		e := newEnv(t)
		err := e.svc.Record(t.Context(), e.actor, app.Entry{
			Action: "sale.deleted", Details: map[string]string{"total": "1.00"},
		})
		require.ErrorIs(t, err, domain.ErrUnknownAction)
		require.Empty(t, e.events.stored)
	})

	t.Run("devuelve el error del registro para que la operación se revierta", func(t *testing.T) {
		e := newEnv(t)
		e.events.fail = errors.New("db down")
		err := e.svc.Record(t.Context(), e.actor, app.Entry{
			Action: domain.ActionModuleEnabled, Details: map[string]string{"module": "variants"},
		})
		require.ErrorContains(t, err, "db down")
		require.Zero(t, e.counter(t, "cuadra.audit.events", "action", "module.enabled"))
	})
}
