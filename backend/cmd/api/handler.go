package main

import (
	"log/slog"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// newHandler builds the API's routes and middleware chain. otelhttp is the
// outermost layer so every log written while serving a request carries its
// trace context. db backs the readiness probe.
func newHandler(log *slog.Logger, tp trace.TracerProvider, db httpx.Pinger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", httpx.Healthz)
	mux.HandleFunc("GET /readyz", httpx.Readyz(log, db))

	handler := httpx.RequestID(httpx.Logging(log)(httpx.Recover(log)(mux)))
	traced := otelhttp.NewHandler(handler, "http.server", otelhttp.WithTracerProvider(tp))
	return withRoutePattern(mux, traced)
}

// withRoutePattern resolves the route pattern before next runs. otelhttp names
// spans and sets http.route from r.Pattern, but the mux only sets it on its own
// copy of the request, which middleware cloning the request hides from otelhttp.
func withRoutePattern(mux *http.ServeMux, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, r.Pattern = mux.Handler(r)
		next.ServeHTTP(w, r)
	})
}
