// Package logger builds the application's structured logger.
package logger

import (
	"context"
	"io"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

// New returns a slog logger that writes JSON to w and sends every record to
// the global OpenTelemetry LoggerProvider (a no-op when telemetry is disabled).
// Each record gets the attributes from extractors. The JSON output also gets
// trace_id and span_id; the OTel bridge already sends the trace context as part
// of the log record, so adding it as attributes there would duplicate it.
func New(w io.Writer, level slog.Level, name string, extractors ...ContextExtractor) *slog.Logger {
	return slog.New(slog.NewMultiHandler(
		NewContextHandler(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}), extractors...),
		&contextHandler{next: minLevel{Handler: otelslog.NewHandler(name), level: level}, extractors: extractors},
	))
}

// minLevel drops records below level; the OTel bridge has no level option.
type minLevel struct {
	slog.Handler
	level slog.Level
}

func (h minLevel) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level && h.Handler.Enabled(ctx, level)
}

func (h minLevel) WithAttrs(attrs []slog.Attr) slog.Handler {
	return minLevel{Handler: h.Handler.WithAttrs(attrs), level: h.level}
}

func (h minLevel) WithGroup(name string) slog.Handler {
	return minLevel{Handler: h.Handler.WithGroup(name), level: h.level}
}
