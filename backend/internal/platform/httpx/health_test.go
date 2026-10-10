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
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx/healthapi"
)

// pingerFunc adapts a function to httpx.Pinger.
type pingerFunc func(context.Context) error

func (f pingerFunc) Ping(ctx context.Context) error { return f(ctx) }

// serveHealth sends a GET to path through the generated health API.
func serveHealth(t *testing.T, db httpx.Pinger, path string) *httptest.ResponseRecorder {
	t.Helper()
	mapper := httpx.NewErrorMapper(discardLogger())
	strict := healthapi.NewStrictHandlerWithOptions(httpx.NewHealth(discardLogger(), db), nil, healthapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  mapper.RequestError,
		ResponseErrorHandlerFunc: mapper.ResponseError,
	})
	rec := httptest.NewRecorder()
	healthapi.Handler(strict).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestHealthz(t *testing.T) {
	t.Run("healthz responde 200 con JSON status ok", func(t *testing.T) {
		rec := serveHealth(t, nil, "/healthz")

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, map[string]string{"status": "ok"}, body)
	})
}

func TestReadyz(t *testing.T) {
	t.Run("readyz responde 200 cuando la BD responde", func(t *testing.T) {
		db := pingerFunc(func(context.Context) error { return nil })

		rec := serveHealth(t, db, "/readyz")

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, map[string]string{"status": "ready"}, body)
	})

	t.Run("readyz responde 503 con el esquema Error cuando la BD falla", func(t *testing.T) {
		db := pingerFunc(func(context.Context) error { return errors.New("connection refused") })

		rec := serveHealth(t, db, "/readyz")

		require.Equal(t, http.StatusServiceUnavailable, rec.Code)
		requireErrorBody(t, rec, "not_ready")
		require.NotContains(t, rec.Body.String(), "connection refused", "no expone detalles internos")
	})

	t.Run("readyz limita el tiempo de espera de la BD", func(t *testing.T) {
		db := pingerFunc(func(ctx context.Context) error {
			_, hasDeadline := ctx.Deadline()
			if !hasDeadline {
				return errors.New("ping without deadline")
			}
			return nil
		})

		rec := serveHealth(t, db, "/readyz")

		require.Equal(t, http.StatusOK, rec.Code)
	})
}
