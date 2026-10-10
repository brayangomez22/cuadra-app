package main

import (
	"context"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/apispec"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx/healthapi"
)

// newHandler builds the API's routes and middleware chain. otelhttp is the
// outermost layer so every log written while serving a request carries its
// trace context. db backs the readiness probe; docs serves /docs and
// /openapi.json (development only).
//
// The operations of api/openapi.yaml live on api, behind the spec validator,
// so a request that breaks the contract never reaches a handler. The docs
// are not part of the contract and live on root, outside the validator.
func newHandler(log *slog.Logger, tp trace.TracerProvider, db httpx.Pinger, docs bool) (http.Handler, error) {
	spec, err := apispec.Load(context.Background())
	if err != nil {
		return nil, err
	}
	validate, err := apispec.NewValidator(spec, log)
	if err != nil {
		return nil, err
	}

	api := http.NewServeMux()
	healthErrors := httpx.NewErrorMapper(log)
	healthapi.HandlerWithOptions(
		healthapi.NewStrictHandlerWithOptions(httpx.NewHealth(log, db), nil, healthapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  healthErrors.RequestError,
			ResponseErrorHandlerFunc: healthErrors.ResponseError,
		}),
		healthapi.StdHTTPServerOptions{BaseRouter: api, ErrorHandlerFunc: healthErrors.RequestError},
	)

	root := http.NewServeMux()
	if docs {
		apispec.RegisterDocs(root)
	}
	root.Handle("/", validate(api))

	handler := httpx.RequestID(httpx.Logging(log)(httpx.Recover(log)(root)))
	traced := otelhttp.NewHandler(handler, "http.server", otelhttp.WithTracerProvider(tp))
	return withRoutePattern(traced, api, root), nil
}

// withRoutePattern resolves the route pattern before next runs. otelhttp names
// spans and sets http.route from r.Pattern, but the mux only sets it on its own
// copy of the request, which middleware cloning the request hides from otelhttp.
// The first mux with a specific route wins: root's catch-all "/" would hide the
// API's patterns.
func withRoutePattern(next http.Handler, muxes ...*http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, mux := range muxes {
			if _, pattern := mux.Handler(r); pattern != "" && pattern != "/" {
				r.Pattern = pattern
				break
			}
		}
		next.ServeHTTP(w, r)
	})
}
