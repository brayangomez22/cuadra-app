package server

import (
	"net/http"
	"net/netip"
	"time"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/ratelimit"
)

// The largest request bodies the API reads (P83): an import's file (10 MB)
// and its form; every other operation sends JSON.
const (
	maxJSONBytes   = 1 << 20
	maxImportBytes = 11 << 20
)

// importUpload is the only operation that takes a file.
const importUpload = "POST /api/v1/imports"

// Edge configures what the API checks of every request before routing it.
type Edge struct {
	// TrustedProxies are the proxies (the cloud's ingress) whose
	// X-Forwarded-For names the client (TRUSTED_PROXIES).
	TrustedProxies []netip.Prefix
	// CORSAllowedOrigins may call the API from a browser
	// (CORS_ALLOWED_ORIGINS); empty means only the API's own origin.
	CORSAllowedOrigins []string
	// HSTS asks browsers to use only HTTPS (production).
	HSTS bool
}

// RateLimits are the requests each client IP and each tenant may make per
// Window, and the businesses one client IP may register per SignupWindow
// (P04). Each replica counts its own (platform/ratelimit): with n replicas
// a client gets up to n times the limit.
type RateLimits struct {
	PerIP        int
	PerTenant    int
	Window       time.Duration
	SignupsPerIP int
	SignupWindow time.Duration
}

// DefaultRateLimits are the API's limits in production: far above what a
// person at a POS or a back office does (a sale is a handful of requests),
// low enough to stop a script hammering the API or a tenant starving the
// others; and a few sign-ups per hour, each of which creates a tenant with
// its template's data.
func DefaultRateLimits() *RateLimits {
	return &RateLimits{PerIP: 600, PerTenant: 3000, Window: time.Minute, SignupsPerIP: 10, SignupWindow: time.Hour}
}

// signupLimiter limits the sign-ups per client IP; nil (no limits) leaves
// them unlimited.
func (l *RateLimits) signupLimiter() *ratelimit.FixedWindow {
	if l == nil {
		return nil
	}
	return ratelimit.NewFixedWindow(l.SignupsPerIP, l.SignupWindow)
}

// edge wraps next with the checks that come before routing: the client's
// IP, the security headers, CORS and the limit per IP.
func edge(next http.Handler, e Edge, limits *RateLimits) http.Handler {
	if limits != nil {
		perIP := ratelimit.NewFixedWindow(limits.PerIP, limits.Window)
		next = httpx.RateLimit(perIP, "ip", func(r *http.Request) (string, bool) {
			ip := httpx.ClientIPFrom(r.Context())
			return ip, ip != ""
		})(next)
	}
	next = httpx.CORS(e.CORSAllowedOrigins)(next)
	next = httpx.SecurityHeaders(e.HSTS)(next)
	return httpx.ClientIP(e.TrustedProxies)(next)
}

// tenantLimit limits the requests of each tenant, once auth.Enforce put the
// principal in the context; anonymous requests only count per IP.
func tenantLimit(limits *RateLimits) func(http.Handler) http.Handler {
	if limits == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	perTenant := ratelimit.NewFixedWindow(limits.PerTenant, limits.Window)
	return httpx.RateLimit(perTenant, "tenant", func(r *http.Request) (string, bool) {
		p, ok := auth.PrincipalFrom(r.Context())
		return p.TenantID.String(), ok
	})
}

// limitBodies caps the body of each request by its operation (an import's
// file or JSON), and then rejects a text the database cannot store
// (httpx.RejectNUL), which reads the capped body.
func limitBodies(api *http.ServeMux) func(http.Handler) http.Handler {
	jsonLimit, fileLimit := httpx.LimitBody(maxJSONBytes), httpx.LimitBody(maxImportBytes)
	return func(next http.Handler) http.Handler {
		next = httpx.RejectNUL(next)
		json, file := jsonLimit(next), fileLimit(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, pattern := api.Handler(r); pattern == importUpload {
				file.ServeHTTP(w, r)
				return
			}
			json.ServeHTTP(w, r)
		})
	}
}
