package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/domain"
)

func TestSendFeedback(t *testing.T) {
	t.Run("guarda el comentario del usuario y encola el aviso al equipo", func(t *testing.T) {
		// Una ferretería y un almacén de calzado.
		for _, in := range []app.FeedbackInput{
			{Message: "No encuentro cómo vender por metros", Screen: "/vender", AppVersion: "1.4.0"},
			{Message: "Me gustaría filtrar por talla", Screen: "/productos", AppVersion: "1.4.0"},
		} {
			e := newEnv(t)

			f, err := e.svc.SendFeedback(t.Context(), e.actor, in)
			require.NoError(t, err)

			require.Equal(t, e.actor.TenantID, f.TenantID())
			require.Equal(t, e.actor.UserID, f.UserID())
			require.Equal(t, in.Message, f.Message())
			require.Equal(t, []any{f.ID()}, []any{e.repo.queued[0]})
			e.span(t, "feedback.SendFeedback")
			require.Equal(t, int64(1), e.counted(t, "cuadra.feedback.received"))
		}
	})

	t.Run("un comentario vacío no se guarda", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.SendFeedback(t.Context(), e.actor, app.FeedbackInput{Message: " ", Screen: "/", AppVersion: "1.4.0"})

		require.ErrorIs(t, err, domain.ErrInvalidMessage)
		require.Empty(t, e.repo.queued)
	})
}
