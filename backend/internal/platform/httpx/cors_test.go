package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

func TestCORS(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := httpx.CORS([]string{"https://app.cuadra.co"})(ok)
	serve := func(method, origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "http://api.cuadra.co/api/v1/sales", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if method == http.MethodOptions {
			req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	t.Run("sin Origin (otro servidor, curl) pasa sin headers de CORS", func(t *testing.T) {
		rec := serve(http.MethodPost, "")
		require.Equal(t, http.StatusOK, rec.Code)
		require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("el mismo origen pasa sin headers de CORS", func(t *testing.T) {
		rec := serve(http.MethodPost, "http://api.cuadra.co")
		require.Equal(t, http.StatusOK, rec.Code)
		require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("un origen permitido recibe sus headers, con credenciales", func(t *testing.T) {
		rec := serve(http.MethodPost, "https://app.cuadra.co")
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "https://app.cuadra.co", rec.Header().Get("Access-Control-Allow-Origin"))
		require.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
		require.Contains(t, rec.Header().Values("Vary"), "Origin")
	})

	t.Run("el preflight de un origen permitido responde 204 sin llegar a la API", func(t *testing.T) {
		rec := serve(http.MethodOptions, "https://app.cuadra.co")
		require.Equal(t, http.StatusNoContent, rec.Code)
		require.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
		require.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	})

	t.Run("otro origen no puede escribir ni hacer preflight", func(t *testing.T) {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions} {
			rec := serve(method, "https://evil.example")
			require.Equal(t, http.StatusForbidden, rec.Code, method)
			require.Contains(t, rec.Body.String(), "origin_not_allowed")
			require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("otro origen puede leer sin headers de CORS: el navegador le oculta la respuesta", func(t *testing.T) {
		rec := serve(http.MethodGet, "https://evil.example")
		require.Equal(t, http.StatusOK, rec.Code)
		require.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	})
}
