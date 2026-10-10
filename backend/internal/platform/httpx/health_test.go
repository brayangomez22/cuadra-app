package httpx_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// pingerFunc adapts a function to httpx.Pinger.
type pingerFunc func(context.Context) error

func (f pingerFunc) Ping(ctx context.Context) error { return f(ctx) }

func TestHealthz(t *testing.T) {
	t.Run("healthz responde 200 con JSON status ok", func(t *testing.T) {
		rec := httptest.NewRecorder()

		httpx.Healthz(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, map[string]string{"status": "ok"}, body)
	})
}

func TestReadyz(t *testing.T) {
	t.Run("readyz responde 200 cuando la BD responde", func(t *testing.T) {
		rec := httptest.NewRecorder()
		db := pingerFunc(func(context.Context) error { return nil })

		httpx.Readyz(discardLogger(), db)(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, map[string]string{"status": "ready"}, body)
	})

	t.Run("readyz responde 503 con el esquema Error cuando la BD falla", func(t *testing.T) {
		rec := httptest.NewRecorder()
		db := pingerFunc(func(context.Context) error { return errors.New("connection refused") })

		httpx.Readyz(discardLogger(), db)(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		require.Equal(t, http.StatusServiceUnavailable, rec.Code)
		var body httpx.ErrorBody
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, "not_ready", body.Error.Code)
		require.NotEmpty(t, body.Error.Message)
		require.NotContains(t, rec.Body.String(), "connection refused", "no expone detalles internos")
	})

	t.Run("readyz limita el tiempo de espera de la BD", func(t *testing.T) {
		rec := httptest.NewRecorder()
		db := pingerFunc(func(ctx context.Context) error {
			_, hasDeadline := ctx.Deadline()
			if !hasDeadline {
				return errors.New("ping without deadline")
			}
			return nil
		})

		httpx.Readyz(discardLogger(), db)(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		require.Equal(t, http.StatusOK, rec.Code)
	})
}
