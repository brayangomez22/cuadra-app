package server

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
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	"github.com/brayangomez22/cuadra-app/backend/internal/server/ports"
)

// readyDB is a database that always answers.
type readyDB struct{}

func (readyDB) Ping(context.Context) error { return nil }

// testSecret signs the access tokens of the tests.
var testSecret = []byte("0123456789abcdef0123456789abcdef")

// fakeModules is a module gate: the modules of Cuadra are known (except
// unknown), and the ones in active are active for every tenant.
type fakeModules struct {
	registry *modules.Registry
	unknown  map[string]bool
	active   map[string]bool
	err      error
	checks   int
}

func newFakeModules(tb testing.TB, active ...string) *fakeModules {
	tb.Helper()
	f := &fakeModules{registry: modulesRegistry(tb), unknown: map[string]bool{}, active: map[string]bool{}}
	for _, id := range active {
		f.active[id] = true
	}
	return f
}

func (f *fakeModules) Known(id string) bool { return !f.unknown[id] && f.registry.Known(id) }

func (f *fakeModules) IsActive(_ context.Context, _ uuid.UUID, id string) (bool, error) {
	f.checks++
	return f.active[id], f.err
}

// modulesRegistry is the registry of Cuadra's modules.
func modulesRegistry(tb testing.TB) *modules.Registry {
	tb.Helper()
	r, err := modules.NewRegistry(modules.Builtin()...)
	require.NoError(tb, err)
	return r
}

// testPermissions is the permission matrix of the core and Cuadra's modules.
func testPermissions(tb testing.TB) *identity.Permissions {
	tb.Helper()
	p, err := identity.NewPermissions(ports.IdentityRegistry{Registry: modulesRegistry(tb)})
	require.NoError(tb, err)
	return p
}

// testConfig builds a handler without modules: tests register stub routes.
// Every module is active.
func testConfig(t *testing.T) handlerConfig {
	t.Helper()
	verifier, err := auth.NewJWT(testSecret, 15*time.Minute)
	require.NoError(t, err)
	return handlerConfig{
		Log:            slog.New(slog.DiscardHandler),
		TracerProvider: noop.NewTracerProvider(),
		DB:             readyDB{},
		Verifier:       verifier,
		Permissions:    testPermissions(t),
		Modules:        newFakeModules(t, modules.Variants),
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

func TestHandlerStartup(t *testing.T) {
	t.Run("un x-module que no está registrado impide arrancar", func(t *testing.T) {
		cfg := testConfig(t)
		gate := newFakeModules(t)
		gate.unknown[modules.Variants] = true
		cfg.Modules = gate

		_, err := newHandler(cfg)

		require.ErrorContains(t, err, modules.Variants)
	})

	t.Run("un x-permission que no está en el registro impide arrancar", func(t *testing.T) {
		cfg := testConfig(t)
		cfg.Permissions = unknownPermission{Permissions: testPermissions(t), permission: "sales:void"}

		_, err := newHandler(cfg)

		require.ErrorContains(t, err, "sales:void")
	})
}

func TestHandlerRequestLog(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())

	// setup logs with the extractors of main.go and mounts a protected stub
	// route, GET /api/v1/me.
	setup := func(t *testing.T) (http.Handler, *bytes.Buffer) {
		t.Helper()
		var buf bytes.Buffer
		cfg := testConfig(t)
		cfg.Log = slog.New(logger.NewContextHandler(slog.NewJSONHandler(&buf, nil), httpx.RequestIDAttrs, auth.PrincipalAttrs))
		cfg.Routes = []func(*http.ServeMux){func(mux *http.ServeMux) {
			mux.HandleFunc("GET /api/v1/me", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
		}}
		handler, err := newHandler(cfg)
		require.NoError(t, err)
		return handler, &buf
	}
	// requestLine returns the "http request" line among the logs in buf.
	requestLine := func(t *testing.T, buf *bytes.Buffer) map[string]any {
		t.Helper()
		for line := range strings.Lines(buf.String()) {
			var entry map[string]any
			require.NoError(t, json.Unmarshal([]byte(line), &entry))
			if entry["msg"] == "http request" {
				return entry
			}
		}
		require.FailNow(t, "no hay línea http request", "logs: %s", buf.String())
		return nil
	}

	t.Run("la línea http request de una request autenticada lleva tenant_id y user_id", func(t *testing.T) {
		handler, buf := setup(t)
		j, err := auth.NewJWT(testSecret, 15*time.Minute)
		require.NoError(t, err)
		tok, _, err := j.Issue(tenantID, userID, "cashier")
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		req.Header.Set("Authorization", "Bearer "+tok)

		handler.ServeHTTP(httptest.NewRecorder(), req)

		entry := requestLine(t, buf)
		require.Equal(t, tenantID.String(), entry["tenant_id"])
		require.Equal(t, userID.String(), entry["user_id"])
	})

	t.Run("la línea http request de una request anónima no lleva tenant_id", func(t *testing.T) {
		handler, buf := setup(t)

		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))

		entry := requestLine(t, buf)
		require.EqualValues(t, http.StatusUnauthorized, entry["status"])
		require.NotContains(t, entry, "tenant_id")
		require.NotContains(t, entry, "user_id")
	})
}
