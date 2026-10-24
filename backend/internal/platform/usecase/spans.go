// Package usecase holds what the use cases of every module share: the span
// of each use case, the metric instruments of a Service and its defaults. It
// imports only the OpenTelemetry API, so app/ packages can use it.
package usecase

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// Spans opens the spans of a module's use cases, named
// "<module>.<UseCase>" (sales.ConfirmSale).
type Spans struct {
	tracer trace.Tracer
	module string
}

// NewSpans builds the Spans of module with the tracer instrumentationName of
// tp; a nil tp is a no-op.
func NewSpans(tp trace.TracerProvider, instrumentationName, module string) Spans {
	if tp == nil {
		tp = tracenoop.NewTracerProvider()
	}
	return Spans{tracer: tp.Tracer(instrumentationName), module: module}
}

// Start opens the span of useCase with attrs.
func (s Spans) Start(ctx context.Context, useCase string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return s.tracer.Start(ctx, s.module+"."+useCase, trace.WithAttributes(attrs...))
}

// StartFor opens the span of useCase run by userID of tenantID: attrs, then
// tenant.id and user.id.
func (s Spans) StartFor(
	ctx context.Context, useCase string, tenantID, userID uuid.UUID, attrs ...attribute.KeyValue,
) (context.Context, trace.Span) {
	attrs = append(attrs,
		attribute.String("tenant.id", tenantID.String()),
		attribute.String("user.id", userID.String()),
	)
	return s.Start(ctx, useCase, attrs...)
}

// End records err on the span, when there is one, and ends it.
func End(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

// Logger returns l, or a logger that discards everything when l is nil.
func Logger(l *slog.Logger) *slog.Logger {
	if l == nil {
		return slog.New(slog.DiscardHandler)
	}
	return l
}
