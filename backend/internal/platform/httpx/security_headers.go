package httpx

import (
	"net/http"
	"strings"
)

// SecurityHeaders sets the headers every response of a JSON API should
// carry (OWASP ASVS 14.4): no MIME sniffing, no framing, a Content Security
// Policy that loads nothing, no referrer, no caching of responses that hold
// business data and, with hsts (production, behind TLS), Strict Transport
// Security. The development docs (/docs, /openapi.json) load their own
// scripts, so they get every header but the CSP. The web app's own headers
// (its CSP) are set where it is served.
func SecurityHeaders(hsts bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Cross-Origin-Resource-Policy", "same-origin")
			h.Set("Cache-Control", "no-store")
			if !isDocs(r.URL.Path) {
				h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			}
			if hsts {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isDocs(path string) bool {
	return path == "/openapi.json" || path == "/docs" || strings.HasPrefix(path, "/docs/")
}
