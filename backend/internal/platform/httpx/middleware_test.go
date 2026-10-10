package httpx_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/logger"
)

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func TestRecover(t *testing.T) {
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("contraseña=secreta123")
	})

	t.Run("recover convierte un panic en 500 con el formato de error estándar", func(t *testing.T) {
		rec := httptest.NewRecorder()

		httpx.Recover(discardLogger())(panicking).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		var body struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, "internal_error", body.Error.Code)
		require.NotEmpty(t, body.Error.Message)
	})

	t.Run("recover no filtra el mensaje del panic al cliente", func(t *testing.T) {
		rec := httptest.NewRecorder()

		httpx.Recover(discardLogger())(panicking).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		require.NotContains(t, rec.Body.String(), "secreta123")
	})
}

func TestRequestID(t *testing.T) {
	var seen string
	handler := httpx.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = httpx.RequestIDFromContext(r.Context())
	}))

	t.Run("request id genera uno nuevo si no viene en la cabecera", func(t *testing.T) {
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		require.NotEmpty(t, seen)
		require.Equal(t, seen, rec.Header().Get(httpx.RequestIDHeader))

		first := seen
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
		require.NotEqual(t, first, seen, "cada request debe tener un ID distinto")
	})

	t.Run("request id respeta el X-Request-ID entrante válido", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(httpx.RequestIDHeader, "abc-123_XYZ.9")

		handler.ServeHTTP(rec, req)

		require.Equal(t, "abc-123_XYZ.9", seen)
		require.Equal(t, "abc-123_XYZ.9", rec.Header().Get(httpx.RequestIDHeader))
	})

	t.Run("request id descarta un X-Request-ID inválido o demasiado largo", func(t *testing.T) {
		for _, incoming := range []string{"id con espacios", "<script>", "id\nfalso", strings.Repeat("a", 65)} {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(httpx.RequestIDHeader, incoming)

			handler.ServeHTTP(httptest.NewRecorder(), req)

			require.NotEmpty(t, seen, "entrante %q", incoming)
			require.NotEqual(t, incoming, seen, "entrante %q", incoming)
		}
	})
}

func TestLogging(t *testing.T) {
	t.Run("logging registra método, ruta, status y duración con el request_id", func(t *testing.T) {
		var buf bytes.Buffer
		log := slog.New(logger.NewContextHandler(slog.NewJSONHandler(&buf, nil), httpx.RequestIDAttrs))
		handler := httpx.RequestID(httpx.Logging(log)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		})))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/algo?token=no-se-loguea", nil)
		req.Header.Set(httpx.RequestIDHeader, "req-42")

		handler.ServeHTTP(httptest.NewRecorder(), req)

		var entry map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
		require.Equal(t, "POST", entry["method"])
		require.Equal(t, "/api/v1/algo", entry["path"])
		require.EqualValues(t, http.StatusTeapot, entry["status"])
		require.Contains(t, entry, "duration_ms")
		require.Equal(t, "req-42", entry["request_id"])
		require.NotContains(t, buf.String(), "no-se-loguea")
	})
}
