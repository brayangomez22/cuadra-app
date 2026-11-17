package httpx

import (
	"math"
	"net/http"
	"strconv"
	"time"
)

// Limiter counts attempts per key (platform/ratelimit, or a shared store if
// a measurement asks for one).
type Limiter interface {
	Allow(key string) (allowed bool, retryAfter time.Duration)
}

// RateLimit answers 429 rate_limited, with Retry-After, to a request whose
// key (its client IP, its tenant...) went over limiter's limit. A request
// without key (keyOf says false) is not counted. scope prefixes the key
// ("ip", "tenant"), so two limits never share a count.
func RateLimit(limiter Limiter, scope string, keyOf func(*http.Request) (string, bool)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, ok := keyOf(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			allowed, retry := limiter.Allow(scope + ":" + key)
			if allowed {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retry.Seconds()))))
			WriteError(w, http.StatusTooManyRequests, "rate_limited",
				"Demasiadas solicitudes. Espere un momento e intente de nuevo.")
		})
	}
}
