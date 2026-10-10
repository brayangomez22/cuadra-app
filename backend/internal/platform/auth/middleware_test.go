package auth_test

import (
	"bytes"
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

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/logger"
)

// can is a tiny permission matrix: admins may write the catalog.
func can(role, permission string) bool {
	return role == "admin" && permission == "catalog:write"
}

// reached records whether a request got through and with which principal.
type reached struct {
	called    bool
	principal auth.Principal
	ok        bool
}

func (r *reached) handler(log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.called = true
		r.principal, r.ok = auth.PrincipalFrom(req.Context())
		log.InfoContext(req.Context(), "handler reached")
		w.WriteHeader(http.StatusNoContent)
	})
}

type harness struct {
	jwt   *auth.JWT
	clock *clock
	log   *slog.Logger
	logs  *bytes.Buffer
	spans *tracetest.SpanRecorder
	tp    *sdktrace.TracerProvider
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	j, c := newJWT(t)
	var buf bytes.Buffer
	spans := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans))
	t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })
	return &harness{
		jwt: j, clock: c, logs: &buf, spans: spans, tp: tp,
		log: slog.New(logger.NewContextHandler(slog.NewJSONHandler(&buf, nil), auth.PrincipalAttrs)),
	}
}

// serve runs req through h inside a server span, like otelhttp would.
func (h *harness) serve(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	ctx, span := h.tp.Tracer("test").Start(req.Context(), "http.server")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req.WithContext(ctx))
	span.End()
	return rec
}

func (h *harness) token(t *testing.T, tenantID, userID uuid.UUID, role string) string {
	t.Helper()
	token, _, err := h.jwt.Issue(tenantID, userID, role)
	require.NoError(t, err)
	return token
}

func requireError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, rec.Code)
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, code, body.Error.Code)
	require.NotEmpty(t, body.Error.Message)
}

func request(token string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestRequireAuth(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())

	t.Run("rechaza request sin token con 401", func(t *testing.T) {
		h := newHarness(t)
		var r reached

		rec := h.serve(auth.RequireAuth(h.jwt, h.log)(r.handler(h.log)), request(""))

		requireError(t, rec, http.StatusUnauthorized, "unauthorized")
		require.Equal(t, "Bearer", rec.Header().Get("WWW-Authenticate"))
		require.False(t, r.called)
	})

	t.Run("rechaza un esquema distinto de Bearer", func(t *testing.T) {
		h := newHarness(t)
		var r reached
		req := request("")
		req.Header.Set("Authorization", "Basic "+h.token(t, tenantID, userID, "admin"))

		rec := h.serve(auth.RequireAuth(h.jwt, h.log)(r.handler(h.log)), req)

		requireError(t, rec, http.StatusUnauthorized, "unauthorized")
		require.False(t, r.called)
	})

	t.Run("rechaza token vencido con 401", func(t *testing.T) {
		h := newHarness(t)
		var r reached
		token := h.token(t, tenantID, userID, "admin")
		h.clock.now = h.clock.now.Add(16 * time.Minute)

		rec := h.serve(auth.RequireAuth(h.jwt, h.log)(r.handler(h.log)), request(token))

		requireError(t, rec, http.StatusUnauthorized, "unauthorized")
		require.False(t, r.called)
	})

	t.Run("rechaza token alterado con 401 y no lo registra en los logs", func(t *testing.T) {
		h := newHarness(t)
		var r reached
		token := h.token(t, tenantID, userID, "admin") + "x"

		rec := h.serve(auth.RequireAuth(h.jwt, h.log)(r.handler(h.log)), request(token))

		requireError(t, rec, http.StatusUnauthorized, "unauthorized")
		require.False(t, r.called)
		require.NotContains(t, h.logs.String(), token[:40])
	})

	t.Run("pone tenant, usuario y rol en el contexto y como atributos del span", func(t *testing.T) {
		h := newHarness(t)
		var r reached

		rec := h.serve(auth.RequireAuth(h.jwt, h.log)(r.handler(h.log)), request(h.token(t, tenantID, userID, "admin")))

		require.Equal(t, http.StatusNoContent, rec.Code)
		require.True(t, r.ok)
		require.Equal(t, auth.Principal{TenantID: tenantID, UserID: userID, Role: "admin"}, r.principal)

		spans := h.spans.Ended()
		require.Len(t, spans, 1)
		attrs := spans[0].Attributes()
		require.Contains(t, attrs, attribute.String("tenant.id", tenantID.String()))
		require.Contains(t, attrs, attribute.String("user.id", userID.String()))
		require.Contains(t, attrs, attribute.String("user.role", "admin"))
	})

	t.Run("los logs del handler llevan tenant_id y user_id", func(t *testing.T) {
		h := newHarness(t)
		var r reached

		h.serve(auth.RequireAuth(h.jwt, h.log)(r.handler(h.log)), request(h.token(t, tenantID, userID, "admin")))

		var entry map[string]any
		require.NoError(t, json.Unmarshal(h.logs.Bytes(), &entry))
		require.Equal(t, "handler reached", entry["msg"])
		require.Equal(t, tenantID.String(), entry["tenant_id"])
		require.Equal(t, userID.String(), entry["user_id"])
	})
}

func TestRequirePermission(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())

	t.Run("responde 403 si el rol no tiene el permiso", func(t *testing.T) {
		h := newHarness(t)
		var r reached
		handler := auth.RequireAuth(h.jwt, h.log)(auth.RequirePermission(can, "catalog:write", h.log)(r.handler(h.log)))

		rec := h.serve(handler, request(h.token(t, tenantID, userID, "cashier")))

		requireError(t, rec, http.StatusForbidden, "forbidden")
		require.False(t, r.called)
	})

	t.Run("deja pasar si el rol tiene el permiso", func(t *testing.T) {
		h := newHarness(t)
		var r reached
		handler := auth.RequireAuth(h.jwt, h.log)(auth.RequirePermission(can, "catalog:write", h.log)(r.handler(h.log)))

		rec := h.serve(handler, request(h.token(t, tenantID, userID, "admin")))

		require.Equal(t, http.StatusNoContent, rec.Code)
		require.True(t, r.called)
	})

	t.Run("sin usuario autenticado responde 401", func(t *testing.T) {
		h := newHarness(t)
		var r reached

		rec := h.serve(auth.RequirePermission(can, "catalog:write", h.log)(r.handler(h.log)), request(""))

		requireError(t, rec, http.StatusUnauthorized, "unauthorized")
		require.False(t, r.called)
	})
}

func TestEnforce(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	policy := func(r *http.Request) (auth.Requirement, bool) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/auth/login":
			return auth.Requirement{Public: true}, true
		case "GET /api/v1/me":
			return auth.Requirement{}, true
		case "POST /api/v1/products":
			return auth.Requirement{Permission: "catalog:write"}, true
		}
		return auth.Requirement{}, false
	}
	serve := func(t *testing.T, h *harness, operation, token string) (*httptest.ResponseRecorder, *reached) {
		t.Helper()
		var r reached
		method, path, _ := strings.Cut(operation, " ")
		req := httptest.NewRequest(method, path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		return h.serve(auth.Enforce(policy, h.jwt, can, h.log)(r.handler(h.log)), req), &r
	}

	t.Run("una operación pública no pide token", func(t *testing.T) {
		h := newHarness(t)
		rec, r := serve(t, h, "POST /api/v1/auth/login", "")
		require.Equal(t, http.StatusNoContent, rec.Code)
		require.True(t, r.called)
	})

	t.Run("una operación sin permiso declarado exige solo el token", func(t *testing.T) {
		h := newHarness(t)
		rec, _ := serve(t, h, "GET /api/v1/me", "")
		requireError(t, rec, http.StatusUnauthorized, "unauthorized")

		rec, r := serve(t, h, "GET /api/v1/me", h.token(t, tenantID, userID, "cashier"))
		require.Equal(t, http.StatusNoContent, rec.Code)
		require.True(t, r.called)
	})

	t.Run("una operación con permiso exige token y permiso", func(t *testing.T) {
		h := newHarness(t)
		rec, _ := serve(t, h, "POST /api/v1/products", "")
		requireError(t, rec, http.StatusUnauthorized, "unauthorized")

		rec, _ = serve(t, h, "POST /api/v1/products", h.token(t, tenantID, userID, "cashier"))
		requireError(t, rec, http.StatusForbidden, "forbidden")

		rec, r := serve(t, h, "POST /api/v1/products", h.token(t, tenantID, userID, "admin"))
		require.Equal(t, http.StatusNoContent, rec.Code)
		require.True(t, r.called)
	})

	t.Run("una ruta desconocida exige token", func(t *testing.T) {
		h := newHarness(t)
		rec, r := serve(t, h, "GET /api/v1/internal", "")
		requireError(t, rec, http.StatusUnauthorized, "unauthorized")
		require.False(t, r.called)
	})
}
