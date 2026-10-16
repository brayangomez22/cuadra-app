package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func newFakeModules(t testing.TB, active ...string) *fakeModules {
	f := &fakeModules{registry: modulesRegistry(t), unknown: map[string]bool{}, active: map[string]bool{}}
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
func modulesRegistry(t testing.TB) *modules.Registry {
	t.Helper()
	r, err := modules.NewRegistry(modules.Builtin()...)
	require.NoError(t, err)
	return r
}

// testPermissions is the permission matrix of the core and Cuadra's modules.
func testPermissions(t testing.TB) *identity.Permissions {
	t.Helper()
	p, err := identity.NewPermissions(identityRegistry{modulesRegistry(t)})
	require.NoError(t, err)
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

func TestHandlerModules(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())
	const variantsPath = "/api/v1/products/01920000-0000-7000-8000-000000000001/variants"
	const validVariant = `{"sku": "CAM-M", "cost": "10000.00", "price": "25000.00", "axis_values": {"talla": "M"}}`

	// setup mounts a stub of addProductVariant, which belongs to the
	// variants module.
	setup := func(t *testing.T, gate *fakeModules) (http.Handler, *bool) {
		t.Helper()
		reached := false
		cfg := testConfig(t)
		cfg.Modules = gate
		cfg.Routes = []func(*http.ServeMux){func(mux *http.ServeMux) {
			mux.HandleFunc("POST /api/v1/products/{productId}/variants", func(w http.ResponseWriter, _ *http.Request) {
				reached = true
				w.WriteHeader(http.StatusNoContent)
			})
		}}
		handler, err := newHandler(cfg)
		require.NoError(t, err)
		return handler, &reached
	}
	post := func(t *testing.T, handler http.Handler, role, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, variantsPath, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if role != "" {
			j, err := auth.NewJWT(testSecret, 15*time.Minute)
			require.NoError(t, err)
			tok, _, err := j.Issue(tenantID, uuid.Must(uuid.NewV7()), role)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	errorCode := func(t *testing.T, rec *httptest.ResponseRecorder) string {
		t.Helper()
		var body httpx.ErrorBody
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "body: %s", rec.Body.String())
		return body.Error.Code
	}

	t.Run("una ruta de un módulo inactivo responde 404", func(t *testing.T) {
		handler, reached := setup(t, newFakeModules(t))

		rec := post(t, handler, "admin", validVariant)

		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Equal(t, "not_found", errorCode(t, rec))
		require.False(t, *reached)
	})

	t.Run("una ruta de un módulo activo llega al handler", func(t *testing.T) {
		handler, reached := setup(t, newFakeModules(t, modules.Variants))

		rec := post(t, handler, "admin", validVariant)

		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.True(t, *reached)
	})

	t.Run("el 404 del módulo inactivo llega antes de validar el body", func(t *testing.T) {
		handler, _ := setup(t, newFakeModules(t))

		rec := post(t, handler, "admin", `{"sku": ""}`)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("sin token responde 401 antes de revisar el módulo", func(t *testing.T) {
		gate := newFakeModules(t)
		handler, _ := setup(t, gate)

		rec := post(t, handler, "", validVariant)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Zero(t, gate.checks)
	})

	t.Run("sin el permiso responde 403 antes de revisar el módulo", func(t *testing.T) {
		gate := newFakeModules(t)
		handler, _ := setup(t, gate)

		rec := post(t, handler, "cashier", validVariant)

		require.Equal(t, http.StatusForbidden, rec.Code)
		require.Zero(t, gate.checks)
	})

	t.Run("si no se puede consultar el módulo responde 500", func(t *testing.T) {
		gate := newFakeModules(t)
		gate.err = errors.New("db down")
		handler, reached := setup(t, gate)

		rec := post(t, handler, "admin", validVariant)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.Equal(t, "internal_error", errorCode(t, rec))
		require.False(t, *reached)
	})

	t.Run("las rutas del core no consultan los módulos", func(t *testing.T) {
		gate := newFakeModules(t)
		cfg := testConfig(t)
		cfg.Modules = gate
		handler, err := newHandler(cfg)
		require.NoError(t, err)

		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

		require.Zero(t, gate.checks)
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

// unknownPermission is a permission matrix that does not register one
// permission.
type unknownPermission struct {
	*identity.Permissions
	permission string
}

func (u unknownPermission) Known(p string) bool { return p != u.permission && u.Permissions.Known(p) }

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

func TestIdentityRegistry(t *testing.T) {
	t.Run("pasa a identity los permisos de cada módulo con sus roles", func(t *testing.T) {
		registry, err := modules.NewRegistry(
			modules.Definition{ID: "variants", Name: "Variantes"},
			modules.Definition{
				ID: "lots_expiry", Name: "Lotes y vencimientos", DependsOn: []string{"variants"},
				Permissions: []modules.Permission{{ID: "lots:manage", Roles: []string{"admin", "warehouse"}}},
			},
		)
		require.NoError(t, err)

		permissions, err := identity.NewPermissions(identityRegistry{registry})

		require.NoError(t, err)
		require.True(t, permissions.Known("lots:manage"))
		require.True(t, permissions.Authorize("warehouse", "lots:manage"))
		require.False(t, permissions.Authorize("cashier", "lots:manage"))
		require.True(t, permissions.Authorize("cashier", "sales:create"))
	})
}
