package httpadapter_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

var cashier = auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "cashier"}

type stubUseCases struct {
	actor app.Actor
	in    app.FeedbackInput
	err   error
}

func (s *stubUseCases) SendFeedback(_ context.Context, a app.Actor, in app.FeedbackInput) (*domain.Feedback, error) {
	s.actor, s.in = a, in
	if s.err != nil {
		return nil, s.err
	}
	return domain.NewFeedback(a.TenantID, a.UserID, in.Message, in.Screen, in.AppVersion, time.Now())
}

func send(t *testing.T, uc *stubUseCases, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, slog.New(slog.DiscardHandler))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithPrincipal(req.Context(), cashier))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestSendFeedback(t *testing.T) {
	t.Run("guarda el comentario del usuario del token", func(t *testing.T) {
		uc := &stubUseCases{}

		rec := send(t, uc, `{"message": "No encuentro el botón de abono", "screen": "/cartera", "app_version": "1.4.0"}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var body struct{ ID uuid.UUID }
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.NotEqual(t, uuid.Nil, body.ID)
		require.Equal(t, app.Actor{TenantID: cashier.TenantID, UserID: cashier.UserID}, uc.actor)
		require.Equal(t, app.FeedbackInput{Message: "No encuentro el botón de abono", Screen: "/cartera", AppVersion: "1.4.0"}, uc.in)
	})

	t.Run("un comentario vacío responde 422", func(t *testing.T) {
		rec := send(t, &stubUseCases{err: domain.ErrInvalidMessage}, `{"message": " ", "screen": "/", "app_version": "1"}`)

		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		require.Contains(t, rec.Body.String(), `"invalid_feedback"`)
	})
}
