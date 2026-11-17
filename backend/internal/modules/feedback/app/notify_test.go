package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/app"
)

func TestNotifyFeedback(t *testing.T) {
	send := func(t *testing.T, e *env) uuid.UUID {
		t.Helper()
		f, err := e.svc.SendFeedback(t.Context(), e.actor, app.FeedbackInput{
			Message: "El cierre de caja no cuadra con los billetes", Screen: "/caja", AppVersion: "1.4.0",
		})
		require.NoError(t, err)
		return f.ID()
	}

	t.Run("le avisa al equipo una sola vez", func(t *testing.T) {
		e := newEnv(t)
		id := send(t, e)

		require.NoError(t, e.svc.NotifyFeedback(t.Context(), e.actor.TenantID, id))
		require.NoError(t, e.svc.NotifyFeedback(t.Context(), e.actor.TenantID, id), "un reintento no avisa dos veces")

		require.Len(t, e.notifier.notices, 1)
		require.Equal(t, app.Notice{
			FeedbackID: id, TenantID: e.actor.TenantID, Screen: "/caja", AppVersion: "1.4.0",
			Message: "El cierre de caja no cuadra con los billetes", CreatedAt: now,
		}, e.notifier.notices[0])
		e.span(t, "feedback.NotifyFeedback")
	})

	t.Run("si el aviso falla devuelve el error para reintentar y no lo marca", func(t *testing.T) {
		e := newEnv(t)
		id := send(t, e)
		e.notifier.fail = errWebhookDown

		require.ErrorIs(t, e.svc.NotifyFeedback(t.Context(), e.actor.TenantID, id), errWebhookDown)

		e.notifier.fail = nil
		require.NoError(t, e.svc.NotifyFeedback(t.Context(), e.actor.TenantID, id))
		require.Len(t, e.notifier.notices, 1)
	})

	t.Run("un comentario que no existe no se reintenta", func(t *testing.T) {
		e := newEnv(t)
		require.NoError(t, e.svc.NotifyFeedback(t.Context(), e.actor.TenantID, uuid.Must(uuid.NewV7())))
		require.Empty(t, e.notifier.notices)
	})
}
