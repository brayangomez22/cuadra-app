package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

func TestSecurityHeaders(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	t.Run("cada respuesta lleva los headers de seguridad de una API JSON", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpx.SecurityHeaders(true)(ok).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))

		h := rec.Header()
		require.Equal(t, "nosniff", h.Get("X-Content-Type-Options"))
		require.Equal(t, "DENY", h.Get("X-Frame-Options"))
		require.Equal(t, "default-src 'none'; frame-ancestors 'none'", h.Get("Content-Security-Policy"))
		require.Equal(t, "no-referrer", h.Get("Referrer-Policy"))
		require.Equal(t, "same-origin", h.Get("Cross-Origin-Resource-Policy"))
		require.Equal(t, "no-store", h.Get("Cache-Control"))
		require.Equal(t, "max-age=31536000; includeSubDomains", h.Get("Strict-Transport-Security"))
	})

	t.Run("sin HSTS fuera de producción", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpx.SecurityHeaders(false)(ok).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))
		require.Empty(t, rec.Header().Get("Strict-Transport-Security"))
		require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	})

	t.Run("la documentación de desarrollo no lleva la CSP de la API", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpx.SecurityHeaders(false)(ok).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs", nil))
		require.Empty(t, rec.Header().Get("Content-Security-Policy"))
		require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	})
}
