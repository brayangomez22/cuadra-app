package main

import (
	"bytes"
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
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/logger"
)

// readyDB is a database that always answers.
type readyDB struct{}

func (readyDB) Ping(context.Context) error { return nil }

// testSecret signs the access tokens of the tests.
var testSecret = []byte("0123456789abcdef0123456789abcdef")

// testConfig builds a handler without modules: tests register stub routes.
func testConfig(t *testing.T) handlerConfig {
	t.Helper()
	verifier, err := auth.NewJWT(testSecret, 15*time.Minute)
	require.NoError(t, err)
	return handlerConfig{
		Log:            slog.New(slog.DiscardHandler),
		TracerProvider: noop.NewTracerProvider(),
		DB:             readyDB{},
		Verifier:       verifier,
		Authorize:      identity.Authorize,
	}
}

func TestHandlerTelemetry(t *testing.T) {
	setup := func(t *testing.T) (http.Handler, *tracetest.SpanRecorder, *bytes.Buffer) {
		t.Helper()
		recorder := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
		t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })
		var buf bytes.Buffer
		log := slog.New(logger.NewContextHandler(slog.NewJSONHandler(&buf, nil), httpx.RequestIDAttrs))
		cfg := testConfig(t)
		cfg.Log, cfg.TracerProvider = log, tp
		handler, err := newHandler(cfg)
		require.NoError(t, err)
		return handler, recorder, &buf
	}

	t.Run("una request a /healthz produce un span con la ruta y el status", func(t *testing.T) {
		handler, recorder, _ := setup(t)

		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

		spans := recorder.Ended()
		require.Len(t, spans, 1)
		require.Equal(t, "GET /healthz", spans[0].Name())
		attrs := map[attribute.Key]attribute.Value{}
		for _, kv := range spans[0].Attributes() {
			attrs[kv.Key] = kv.Value
		}
		require.Equal(t, "/healthz", attrs["http.route"].AsString())
		require.EqualValues(t, http.StatusOK, attrs["http.response.status_code"].AsInt64())
	})

	t.Run("los logs de una request llevan el trace_id de su span", func(t *testing.T) {
		handler, recorder, buf := setup(t)
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set(httpx.RequestIDHeader, "req-42")

		handler.ServeHTTP(httptest.NewRecorder(), req)

		spans := recorder.Ended()
		require.Len(t, spans, 1)
		var entry map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
		require.Equal(t, "http request", entry["msg"])
		require.Equal(t, spans[0].SpanContext().TraceID().String(), entry["trace_id"])
		require.Equal(t, "req-42", entry["request_id"])
	})
}

func TestHandlerRouting(t *testing.T) {
	serve := func(t *testing.T, docs bool, target string) *httptest.ResponseRecorder {
		t.Helper()
		cfg := testConfig(t)
		cfg.Docs = docs
		handler, err := newHandler(cfg)
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		return rec
	}
	errorCode := func(t *testing.T, rec *httptest.ResponseRecorder) string {
		t.Helper()
		var body httpx.ErrorBody
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "body: %s", rec.Body.String())
		return body.Error.Code
	}

	t.Run("/healthz pasa la validación del spec y responde 200", func(t *testing.T) {
		rec := serve(t, false, "/healthz")

		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"status": "ok"}`, rec.Body.String())
	})

	t.Run("/readyz pasa la validación del spec y responde 200", func(t *testing.T) {
		rec := serve(t, false, "/readyz")

		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"status": "ready"}`, rec.Body.String())
	})

	t.Run("una ruta que no está en el spec responde 404 con el formato estándar", func(t *testing.T) {
		rec := serve(t, false, "/api/v1/nope")

		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Equal(t, "not_found", errorCode(t, rec))
	})

	t.Run("/docs está disponible en development", func(t *testing.T) {
		rec := serve(t, true, "/docs")

		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
	})

	t.Run("/docs y /openapi.json no existen fuera de development", func(t *testing.T) {
		for _, target := range []string{"/docs", "/openapi.json"} {
			rec := serve(t, false, target)

			require.Equal(t, http.StatusNotFound, rec.Code, target)
			require.Equal(t, "not_found", errorCode(t, rec), target)
		}
	})
}

func TestHandlerAuth(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())

	// setup mounts stub handlers: GET /api/v1/me (in the contract, protected)
	// and GET /api/v1/internal (on the mux, but not in the contract).
	setup := func(t *testing.T) (http.Handler, *tracetest.SpanRecorder, *auth.Principal) {
		t.Helper()
		recorder := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
		t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })
		var seen auth.Principal
		cfg := testConfig(t)
		cfg.TracerProvider = tp
		cfg.Routes = []func(*http.ServeMux){func(mux *http.ServeMux) {
			mux.HandleFunc("GET /api/v1/me", func(w http.ResponseWriter, r *http.Request) {
				seen, _ = auth.PrincipalFrom(r.Context())
				w.WriteHeader(http.StatusNoContent)
			})
			mux.HandleFunc("GET /api/v1/internal", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
		}}
		handler, err := newHandler(cfg)
		require.NoError(t, err)
		return handler, recorder, &seen
	}
	token := func(t *testing.T) string {
		t.Helper()
		j, err := auth.NewJWT(testSecret, 15*time.Minute)
		require.NoError(t, err)
		tok, _, err := j.Issue(tenantID, userID, "cashier")
		require.NoError(t, err)
		return tok
	}
	errorCode := func(t *testing.T, rec *httptest.ResponseRecorder) string {
		t.Helper()
		var body httpx.ErrorBody
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "body: %s", rec.Body.String())
		return body.Error.Code
	}

	t.Run("una operación protegida sin token responde 401", func(t *testing.T) {
		handler, _, _ := setup(t)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Equal(t, "unauthorized", errorCode(t, rec))
	})

	t.Run("con un token válido llega al handler con el principal y el span lleva el tenant", func(t *testing.T) {
		handler, recorder, seen := setup(t)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		req.Header.Set("Authorization", "Bearer "+token(t))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
		require.Equal(t, auth.Principal{TenantID: tenantID, UserID: userID, Role: "cashier"}, *seen)
		spans := recorder.Ended()
		require.Len(t, spans, 1)
		require.Contains(t, spans[0].Attributes(), attribute.String("tenant.id", tenantID.String()))
	})

	t.Run("una operación pública con body inválido responde 400 sin pedir token", func(t *testing.T) {
		handler, _, _ := setup(t)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"ana@x.co"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "invalid_request", errorCode(t, rec))
	})

	t.Run("una ruta del mux que no está en el contrato exige token", func(t *testing.T) {
		handler, _, _ := setup(t)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/internal", nil))

		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
