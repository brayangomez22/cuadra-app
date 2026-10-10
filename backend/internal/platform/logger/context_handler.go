package logger

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// ContextExtractor returns log attributes carried by ctx (request ID, tenant...).
// Extractors keep this package independent from the packages that own those
// values.
type ContextExtractor func(context.Context) []slog.Attr

// contextHandler adds extracted attributes, and optionally trace context, to
// every record.
type contextHandler struct {
	next       slog.Handler
	extractors []ContextExtractor
	withTrace  bool
}

// NewContextHandler wraps next so every record gets trace_id and span_id (when
// ctx holds a valid span) plus the attributes returned by extractors.
func NewContextHandler(next slog.Handler, extractors ...ContextExtractor) slog.Handler {
	return &contextHandler{next: next, extractors: extractors, withTrace: true}
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); h.withTrace && sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	for _, extract := range h.extractors {
		r.AddAttrs(extract(ctx)...)
	}
	return h.next.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{next: h.next.WithAttrs(attrs), extractors: h.extractors, withTrace: h.withTrace}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{next: h.next.WithGroup(name), extractors: h.extractors, withTrace: h.withTrace}
}
