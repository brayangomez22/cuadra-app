package httpx

import (
	"context"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// OperationKey names a business operation on an HTTP request, so an SLO
// measures it by what the user does (pos.search) instead of by route. Its
// values are few and fixed, safe as a metric attribute.
const OperationKey = attribute.Key("cuadra.operation")

// SetOperation marks the current request as the business operation name: on
// its server span and on the HTTP metrics otelhttp records for it
// (http.server.request.duration). Without otelhttp in ctx it does nothing.
func SetOperation(ctx context.Context, name string) {
	attr := OperationKey.String(name)
	trace.SpanFromContext(ctx).SetAttributes(attr)
	if labeler, ok := otelhttp.LabelerFromContext(ctx); ok {
		labeler.Add(attr)
	}
}
