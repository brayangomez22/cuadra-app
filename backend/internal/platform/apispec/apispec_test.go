package apispec_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/apispec"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// testSpec is a minimal contract with a query parameter and a request body.
// It declares servers and a global security requirement like the real spec.
const testSpec = `
openapi: 3.0.3
info: {title: test, version: "1"}
servers:
  - url: http://localhost:8080
security:
  - bearerAuth: []
paths:
  /items:
    get:
      operationId: listItems
      parameters:
        - name: limit
          in: query
          schema: {type: integer, minimum: 1, maximum: 100}
      responses:
        "200": {description: ok}
    post:
      operationId: createItem
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name: {type: string}
                password: {type: string, minLength: 12}
      responses:
        "201": {description: created}
components:
  securitySchemes:
    bearerAuth: {type: http, scheme: bearer}
`

func TestValidator(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantCode   string // empty when the request must reach the handler
	}{
		{
			name:       "deja pasar una request que cumple el spec",
			method:     http.MethodGet,
			target:     "/items?limit=10",
			wantStatus: http.StatusOK,
		},
		{
			name:       "deja pasar un body que cumple el spec",
			method:     http.MethodPost,
			target:     "/items",
			body:       `{"name": "Tubo PVC"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "rechaza con 400 un query param fuera de rango",
			method:     http.MethodGet,
			target:     "/items?limit=500",
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "rechaza con 400 un query param que no es un número",
			method:     http.MethodGet,
			target:     "/items?limit=diez",
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "rechaza con 400 un body al que le falta un campo requerido",
			method:     http.MethodPost,
			target:     "/items",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "rechaza con 400 un body que no es JSON válido",
			method:     http.MethodPost,
			target:     "/items",
			body:       `{"name":`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "responde 404 con el formato estándar a una ruta que no está en el spec",
			method:     http.MethodGet,
			target:     "/nope",
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
		{
			name:       "responde 405 con el formato estándar a un método que la ruta no tiene",
			method:     http.MethodDelete,
			target:     "/items",
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   "method_not_allowed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, reached := validated(t, discardLogger())
			req := httptest.NewRequest(tt.method, tt.target, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code, "body: %s", rec.Body.String())
			if tt.wantCode == "" {
				require.True(t, *reached, "la request llega al handler")
				return
			}
			require.False(t, *reached, "la request no llega al handler")
			requireErrorBody(t, rec, tt.wantCode)
		})
	}

	t.Run("no exige el token: la autenticación es responsabilidad de RequireAuth", func(t *testing.T) {
		handler, reached := validated(t, discardLogger())
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/items", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		require.True(t, *reached)
	})

	t.Run("el log del rechazo dice qué campo falló sin incluir el valor recibido", func(t *testing.T) {
		var logs bytes.Buffer
		handler, _ := validated(t, slog.New(slog.NewJSONHandler(&logs, nil)))
		req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{"name": "x", "password": "secreto123"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.NotContains(t, rec.Body.String(), "secreto123")
		require.Contains(t, logs.String(), "password")
		require.NotContains(t, logs.String(), "secreto123")
	})
}

func TestLoad(t *testing.T) {
	t.Run("el spec embebido es un OpenAPI válido y describe los probes", func(t *testing.T) {
		spec, err := apispec.Load(t.Context())

		require.NoError(t, err)
		require.NotNil(t, spec.Paths.Find("/healthz"))
		require.NotNil(t, spec.Paths.Find("/readyz"))
	})
}

func TestDocs(t *testing.T) {
	mux := http.NewServeMux()
	apispec.RegisterDocs(mux)

	t.Run("/docs sirve una página HTML que carga el spec", func(t *testing.T) {
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "text/html")
		require.Contains(t, rec.Body.String(), "/openapi.json")
	})

	t.Run("/openapi.json sirve el spec en JSON", func(t *testing.T) {
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		spec, err := openapi3.NewLoader().LoadFromData(rec.Body.Bytes())
		require.NoError(t, err)
		require.NotNil(t, spec.Paths.Find("/healthz"))
	})
}

// validated wraps a handler that answers 200 with the validator built from
// testSpec. reached reports whether the request got through.
func validated(t *testing.T, log *slog.Logger) (http.Handler, *bool) {
	t.Helper()
	spec, err := openapi3.NewLoader().LoadFromData([]byte(testSpec))
	require.NoError(t, err)
	validator, err := apispec.NewValidator(spec, log)
	require.NoError(t, err)
	reached := new(bool)
	return validator(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		*reached = true
		w.WriteHeader(http.StatusOK)
	})), reached
}

func requireErrorBody(t *testing.T, rec *httptest.ResponseRecorder, code string) {
	t.Helper()
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var body httpx.ErrorBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "body: %s", rec.Body.String())
	require.Equal(t, code, body.Error.Code)
	require.NotEmpty(t, body.Error.Message)
}

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }
