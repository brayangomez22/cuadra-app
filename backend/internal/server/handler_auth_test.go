package server

import (
	"encoding/json"
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
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

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
