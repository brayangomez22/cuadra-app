package notify_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/adapters/notify"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/app"
)

func TestWebhook(t *testing.T) {
	notice := app.Notice{
		FeedbackID: uuid.Must(uuid.NewV7()), TenantID: uuid.Must(uuid.NewV7()), Screen: "/vender",
		AppVersion: "1.4.0", Message: "El POS se demora con el lector", CreatedAt: time.Now(),
	}

	t.Run("envía el comentario con la pantalla y la versión", func(t *testing.T) {
		var got map[string]string
		var contentType string
		var decodeErr error
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType = r.Header.Get("Content-Type")
			decodeErr = json.NewDecoder(r.Body).Decode(&got)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		require.NoError(t, notify.NewWebhook(srv.URL).Notify(t.Context(), notice))
		require.NoError(t, decodeErr)
		require.Equal(t, "application/json", contentType)

		require.Contains(t, got["text"], "El POS se demora con el lector")
		require.Contains(t, got["text"], "/vender")
		require.Contains(t, got["text"], "1.4.0")
		require.Equal(t, got["text"], got["content"])
	})

	t.Run("una respuesta de error es un error, para reintentar", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer srv.Close()

		require.ErrorContains(t, notify.NewWebhook(srv.URL).Notify(t.Context(), notice), "502")
	})
}
