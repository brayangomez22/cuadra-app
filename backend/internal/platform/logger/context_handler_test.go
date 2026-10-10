package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/logger"
)

type ctxKey struct{}

func requestIDFromCtx(ctx context.Context) []slog.Attr {
	if id, ok := ctx.Value(ctxKey{}).(string); ok {
		return []slog.Attr{slog.String("request_id", id)}
	}
	return nil
}

// logEntry logs one message through a context handler and decodes the JSON.
func logEntry(ctx context.Context, t *testing.T, build func(*slog.Logger) *slog.Logger) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	log := slog.New(logger.NewContextHandler(slog.NewJSONHandler(&buf, nil), requestIDFromCtx))
	if build != nil {
		log = build(log)
	}

	log.InfoContext(ctx, "hola")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	return entry
}

func TestContextHandler(t *testing.T) {
	t.Run("agrega trace_id y span_id cuando hay un span en el contexto", func(t *testing.T) {
		tp := sdktrace.NewTracerProvider()
		ctx, span := tp.Tracer("test").Start(context.Background(), "op")
		defer span.End()

		entry := logEntry(ctx, t, nil)

		require.Equal(t, span.SpanContext().TraceID().String(), entry["trace_id"])
		require.Equal(t, span.SpanContext().SpanID().String(), entry["span_id"])
	})

	t.Run("omite trace_id y span_id cuando no hay span en el contexto", func(t *testing.T) {
		entry := logEntry(context.Background(), t, nil)

		require.NotContains(t, entry, "trace_id")
		require.NotContains(t, entry, "span_id")
	})

	t.Run("agrega request_id cuando está en el contexto", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ctxKey{}, "req-42")

		entry := logEntry(ctx, t, nil)

		require.Equal(t, "req-42", entry["request_id"])
	})

	t.Run("conserva los atributos agregados con With", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), ctxKey{}, "req-42")

		entry := logEntry(ctx, t, func(l *slog.Logger) *slog.Logger {
			return l.With(slog.String("env", "test"))
		})

		require.Equal(t, "test", entry["env"])
		require.Equal(t, "req-42", entry["request_id"])
	})
}

func TestNew(t *testing.T) {
	t.Run("escribe JSON con trace_id y los atributos de los extractores desde el nivel configurado", func(t *testing.T) {
		var buf bytes.Buffer
		log := logger.New(&buf, slog.LevelInfo, "test", requestIDFromCtx)
		tp := sdktrace.NewTracerProvider()
		ctx, span := tp.Tracer("test").Start(context.WithValue(context.Background(), ctxKey{}, "req-42"), "op")
		defer span.End()

		log.DebugContext(ctx, "no se escribe")
		log.InfoContext(ctx, "hola")

		var entry map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
		require.Equal(t, "hola", entry["msg"])
		require.Equal(t, span.SpanContext().TraceID().String(), entry["trace_id"])
		require.Equal(t, "req-42", entry["request_id"])
	})
}
