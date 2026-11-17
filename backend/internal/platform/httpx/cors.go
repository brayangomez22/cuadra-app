package httpx

import (
	"net/http"
	"slices"
	"strings"
)

// corsMaxAge is how long a browser caches a preflight, in seconds.
const corsMaxAge = "600"

// CORS lets the origins in allowed call the API from a browser, with
// credentials (the refresh cookie), and nobody else: a request from another
// origin that writes (or its preflight) gets 403 origin_not_allowed before
// reaching the API, and one that reads gets no CORS headers, so the browser
// hides the response. Requests without Origin (servers, curl) and from the
// API's own origin pass untouched. With no allowed origins the web app must
// be served from the API's origin (docs/decisiones.md).
func CORS(allowed []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			switch {
			case origin == "" || sameOrigin(r, origin):
				next.ServeHTTP(w, r)
			case slices.Contains(allowed, origin):
				allowOrigin(w, origin)
				if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				next.ServeHTTP(w, r)
			case isSafe(r.Method):
				next.ServeHTTP(w, r)
			default:
				WriteError(w, http.StatusForbidden, "origin_not_allowed",
					"Este origen no tiene permiso para usar la API.")
			}
		})
	}
}

func allowOrigin(w http.ResponseWriter, origin string) {
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", origin)
	h.Set("Access-Control-Allow-Credentials", "true")
	h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
	h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
	h.Set("Access-Control-Expose-Headers", "X-Request-ID, Retry-After")
	h.Set("Access-Control-Max-Age", corsMaxAge)
	h.Add("Vary", "Origin")
}

// sameOrigin tells whether origin is the host the request was sent to.
func sameOrigin(r *http.Request, origin string) bool {
	_, host, ok := strings.Cut(origin, "://")
	return ok && host == r.Host
}

func isSafe(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}
