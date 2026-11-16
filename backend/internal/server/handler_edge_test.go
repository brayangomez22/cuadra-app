package server

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

// edgeHandler mounts stub handlers for GET /api/v1/me and the JSON and
// file uploads, behind the edge of cfg.
func edgeHandler(t *testing.T, edit func(*handlerConfig)) http.Handler {
	t.Helper()
	cfg := testConfig(t)
	cfg.Routes = []func(*http.ServeMux){func(mux *http.ServeMux) {
		ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }
		mux.HandleFunc("GET /api/v1/me", ok)
		mux.HandleFunc("POST /api/v1/customers", ok)
		mux.HandleFunc("POST /api/v1/imports", ok)
	}}
	edit(&cfg)
	h, err := newHandler(cfg)
	require.NoError(t, err)
	return h
}

func ownerToken(t *testing.T, tenantID uuid.UUID) string {
	t.Helper()
	j, err := auth.NewJWT(testSecret, 15*time.Minute)
	require.NoError(t, err)
	tok, _, err := j.Issue(tenantID, uuid.Must(uuid.NewV7()), "owner")
	require.NoError(t, err)
	return "Bearer " + tok
}

func TestHandlerEdge(t *testing.T) {
	t.Run("cada respuesta lleva los headers de seguridad, y HSTS si se pide", func(t *testing.T) {
		h := edgeHandler(t, func(c *handlerConfig) { c.Edge.HSTS = true })
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		require.Equal(t, "default-src 'none'; frame-ancestors 'none'", rec.Header().Get("Content-Security-Policy"))
		require.NotEmpty(t, rec.Header().Get("Strict-Transport-Security"))
	})

	t.Run("una escritura desde otro origen responde 403 antes de la API", func(t *testing.T) {
		h := edgeHandler(t, func(c *handlerConfig) { c.Edge.CORSAllowedOrigins = []string{"https://app.cuadra.co"} })
		req := httptest.NewRequest(http.MethodPost, "/api/v1/customers", strings.NewReader("{}"))
		req.Header.Set("Origin", "https://evil.example")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusForbidden, rec.Code)
		require.Contains(t, rec.Body.String(), "origin_not_allowed")
	})

	t.Run("un JSON de más de 1 MB responde 413; una importación acepta hasta 11 MB", func(t *testing.T) {
		h := edgeHandler(t, func(*handlerConfig) {})
		token := ownerToken(t, uuid.Must(uuid.NewV7()))
		send := func(path string, size int, contentType string) int {
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(strings.Repeat("a", size)))
			req.Header.Set("Authorization", token)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			return rec.Code
		}
		require.Equal(t, http.StatusRequestEntityTooLarge, send("/api/v1/customers", maxJSONBytes+1, "application/json"))
		require.NotEqual(t, http.StatusRequestEntityTooLarge, send("/api/v1/imports", 5<<20, "multipart/form-data; boundary=x"))
		require.Equal(t, http.StatusRequestEntityTooLarge, send("/api/v1/imports", maxImportBytes+1, "multipart/form-data; boundary=x"))
	})

	t.Run("un texto con NUL responde 400 antes de llegar a la base", func(t *testing.T) {
		h := edgeHandler(t, func(*handlerConfig) {})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/customers", strings.NewReader(`{"name":"Ana\u0000"}`))
		req.Header.Set("Authorization", ownerToken(t, uuid.Must(uuid.NewV7())))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Contains(t, rec.Body.String(), "caracteres no válidos", "lo rechaza RejectNUL, no el validador")
	})

	t.Run("pasado el límite por IP responde 429, sin importar el token", func(t *testing.T) {
		h := edgeHandler(t, func(c *handlerConfig) { c.RateLimits = &RateLimits{PerIP: 2, PerTenant: 100, Window: time.Minute} })
		codes := []int{}
		for range 3 {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
			codes = append(codes, rec.Code)
		}
		require.Equal(t, []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests}, codes)
	})

	t.Run("pasado el límite de la empresa responde 429 aunque cambie la IP; otra empresa sigue", func(t *testing.T) {
		h := edgeHandler(t, func(c *handlerConfig) {
			c.RateLimits = &RateLimits{PerIP: 100, PerTenant: 2, Window: time.Minute}
			c.Edge.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
		})
		busy, other := ownerToken(t, uuid.Must(uuid.NewV7())), ownerToken(t, uuid.Must(uuid.NewV7()))
		get := func(token, ip string) int {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
			req.RemoteAddr = "10.0.0.1:443"
			req.Header.Set("X-Forwarded-For", ip)
			req.Header.Set("Authorization", token)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			return rec.Code
		}
		require.Equal(t, http.StatusNoContent, get(busy, "198.51.100.1"))
		require.Equal(t, http.StatusNoContent, get(busy, "198.51.100.2"))
		require.Equal(t, http.StatusTooManyRequests, get(busy, "198.51.100.3"))
		require.Equal(t, http.StatusNoContent, get(other, "198.51.100.3"))
	})
}
