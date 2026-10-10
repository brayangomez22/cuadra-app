package main

import (
	"context"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/apispec"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx/healthapi"
)

// handlerConfig are the dependencies of the API's handler.
type handlerConfig struct {
	Log            *slog.Logger
	TracerProvider trace.TracerProvider
	// DB backs the readiness probe.
	DB httpx.Pinger
	// Docs serves /docs and /openapi.json (development only).
	Docs bool
	// Verifier checks access tokens; Authorize is the role → permission matrix.
	Verifier  auth.Verifier
	Authorize auth.Authorizer
	// Routes mount each module's operations on the API mux.
	Routes []func(*http.ServeMux)
}

// newHandler builds the API's routes and middleware chain. otelhttp is the
// outermost layer so every log written while serving a request carries its
// trace context.
//
// The operations of api/openapi.yaml live on api, behind two gates: first
// auth.Enforce applies the contract's security (a token, and the operation's
// x-permission), then the spec validator rejects requests that break the
// contract. An anonymous caller gets 401 before learning anything about the
// shape of a protected request. The docs are not part of the contract and
// live on root, outside both gates.
func newHandler(cfg handlerConfig) (http.Handler, error) {
	spec, err := apispec.Load(context.Background())
	if err != nil {
		return nil, err
	}
	security, err := apispec.OperationSecurity(spec)
	if err != nil {
		return nil, err
	}
	validate, err := apispec.NewValidator(spec, cfg.Log)
	if err != nil {
		return nil, err
	}

	api := http.NewServeMux()
	healthErrors := httpx.NewErrorMapper(cfg.Log)
	healthapi.HandlerWithOptions(
		healthapi.NewStrictHandlerWithOptions(httpx.NewHealth(cfg.Log, cfg.DB), nil, healthapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  healthErrors.RequestError,
			ResponseErrorHandlerFunc: healthErrors.ResponseError,
		}),
		healthapi.StdHTTPServerOptions{BaseRouter: api, ErrorHandlerFunc: healthErrors.RequestError},
	)
	for _, register := range cfg.Routes {
		register(api)
	}

	enforce := auth.Enforce(contractPolicy(api, security), cfg.Verifier, cfg.Authorize, cfg.Log)

	root := http.NewServeMux()
	if cfg.Docs {
		apispec.RegisterDocs(root)
	}
	root.Handle("/", enforce(validate(api)))

	handler := httpx.RequestID(httpx.Logging(cfg.Log)(httpx.Recover(cfg.Log)(root)))
	traced := otelhttp.NewHandler(handler, "http.server", otelhttp.WithTracerProvider(cfg.TracerProvider))
	return withRoutePattern(traced, api, root), nil
}

// contractPolicy resolves a request's route on api and returns its security
// from the contract. A request no route matches reaches no handler (the
// validator or the mux answer 404/405), so it needs no token; a route on the
// mux that the contract does not describe requires one.
func contractPolicy(api *http.ServeMux, security map[string]apispec.Security) auth.Policy {
	return func(r *http.Request) (auth.Requirement, bool) {
		_, pattern := api.Handler(r)
		if pattern == "" {
			return auth.Requirement{Public: true}, true
		}
		sec, ok := security[pattern]
		return auth.Requirement{Public: sec.Public, Permission: sec.Permission}, ok
	}
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
