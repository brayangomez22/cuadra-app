package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// countingLimiter allows limit attempts per key.
type countingLimiter struct {
	limit int
	seen  map[string]int
}

func (l *countingLimiter) Allow(key string) (bool, time.Duration) {
	l.seen[key]++
	if l.seen[key] > l.limit {
		return false, 42 * time.Second
	}
	return true, 0
}

func TestRateLimit(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	keyOf := func(r *http.Request) (string, bool) {
		k := r.Header.Get("X-Key")
		return k, k != ""
	}

	t.Run("responde 429 con Retry-After al pasar el límite de la llave", func(t *testing.T) {
		limiter := &countingLimiter{limit: 2, seen: map[string]int{}}
		handler := httpx.RateLimit(limiter, "ip", keyOf)(ok)
		serve := func(key string) *httptest.ResponseRecorder {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/sales", nil)
			req.Header.Set("X-Key", key)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			return rec
		}
		require.Equal(t, http.StatusOK, serve("a").Code)
		require.Equal(t, http.StatusOK, serve("a").Code)
		rec := serve("a")
		require.Equal(t, http.StatusTooManyRequests, rec.Code)
		require.Equal(t, "42", rec.Header().Get("Retry-After"))
		require.Contains(t, rec.Body.String(), "rate_limited")
		require.Equal(t, http.StatusOK, serve("b").Code, "cada llave cuenta aparte")
	})

	t.Run("sin llave no cuenta", func(t *testing.T) {
		limiter := &countingLimiter{limit: 0, seen: map[string]int{}}
		rec := httptest.NewRecorder()
		httpx.RateLimit(limiter, "tenant", keyOf)(ok).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Empty(t, limiter.seen)
	})
}
