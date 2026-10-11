package errreport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/errreport"
)

// fakeTransport keeps the events instead of sending them to Sentry.
type fakeTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (f *fakeTransport) Configure(sentry.ClientOptions)        {}
func (f *fakeTransport) Flush(time.Duration) bool              { return true }
func (f *fakeTransport) FlushWithContext(context.Context) bool { return true }
func (f *fakeTransport) Close()                                {}
func (f *fakeTransport) SendEvent(e *sentry.Event) {
	f.mu.Lock()
	f.events = append(f.events, e)
	f.mu.Unlock()
}
func (f *fakeTransport) sent() []*sentry.Event { f.mu.Lock(); defer f.mu.Unlock(); return f.events }

const (
	testDSN    = "https://public@o1.ingest.us.sentry.io/1"
	tenantID   = "0192f0a1-0000-7000-8000-0000000000aa"
	userID     = "0192f0a1-0000-7000-8000-0000000000bb"
	requestID  = "req-123"
	customerNm = "Ana Pérez"
)

type ctxKey struct{}

// principalAttrs plays the role of the auth and request ID extractors: it
// also returns an attribute outside the allowlist, which must not be sent.
func principalAttrs(ctx context.Context) []slog.Attr {
	if ctx.Value(ctxKey{}) == nil {
		return nil
	}
	return []slog.Attr{
		slog.String("tenant_id", tenantID),
		slog.String("user_id", userID),
		slog.String("request_id", requestID),
		slog.String("user_email", "ana@correo.com"),
	}
}

func newReporter(t *testing.T) (*errreport.Reporter, *fakeTransport) {
	t.Helper()
	transport := &fakeTransport{}
	r, err := errreport.New(errreport.Options{
		DSN:         testDSN,
		Environment: "test",
		Release:     "abc123",
		Extractors:  []func(context.Context) []slog.Attr{principalAttrs},
		Transport:   transport,
	})
	require.NoError(t, err)
	return r, transport
}

func TestHandler(t *testing.T) {
	t.Run("reporta registros de nivel Error y no los de Warn o Info", func(t *testing.T) {
		r, transport := newReporter(t)
		log := slog.New(r.Handler(slog.NewJSONHandler(&bytes.Buffer{}, nil)))

		log.Info("venta confirmada")
		log.Warn("telemetría degradada")
		log.Error("unhandled error", slog.Any("error", errors.New("boom")))

		require.Len(t, transport.sent(), 1)
		e := transport.sent()[0]
		require.Equal(t, sentry.LevelError, e.Level)
		require.Equal(t, "unhandled error", e.Message)
		require.Equal(t, "test", e.Environment)
		require.Equal(t, "abc123", e.Release)
	})

	t.Run("sigue escribiendo el log en el handler original", func(t *testing.T) {
		r, _ := newReporter(t)
		var out bytes.Buffer
		log := slog.New(r.Handler(slog.NewJSONHandler(&out, nil)))

		log.Info("venta confirmada")
		log.Error("unhandled error")

		require.Contains(t, out.String(), "venta confirmada")
		require.Contains(t, out.String(), "unhandled error")
	})

	t.Run("adjunta tenant_id, request_id, user.id y la traza desde el contexto", func(t *testing.T) {
		r, transport := newReporter(t)
		log := slog.New(r.Handler(slog.NewJSONHandler(&bytes.Buffer{}, nil)))
		tp := sdktrace.NewTracerProvider()
		ctx, span := tp.Tracer("test").Start(context.WithValue(context.Background(), ctxKey{}, true), "op")
		defer span.End()

		log.ErrorContext(ctx, "unhandled error", slog.Any("error", errors.New("boom")))

		require.Len(t, transport.sent(), 1)
		e := transport.sent()[0]
		require.Equal(t, tenantID, e.Tags["tenant_id"])
		require.Equal(t, requestID, e.Tags["request_id"])
		require.Equal(t, userID, e.User.ID)
		sc := span.SpanContext()
		require.Equal(t, sc.TraceID().String(), e.Tags["trace_id"])
		require.Equal(t, sc.TraceID().String(), e.Contexts["trace"]["trace_id"])
		require.Equal(t, sc.SpanID().String(), e.Contexts["trace"]["span_id"])
	})

	t.Run("no envía los atributos del log ni los que no están en la lista permitida", func(t *testing.T) {
		r, transport := newReporter(t)
		log := slog.New(r.Handler(slog.NewJSONHandler(&bytes.Buffer{}, nil)))
		ctx := context.WithValue(context.Background(), ctxKey{}, true)

		log.ErrorContext(ctx, "unhandled error", slog.String("customer_name", customerNm), slog.Any("error", errors.New("boom")))

		require.Len(t, transport.sent(), 1)
		raw, err := json.Marshal(transport.sent()[0])
		require.NoError(t, err)
		require.NotContains(t, string(raw), customerNm)
		require.NotContains(t, string(raw), "customer_name")
		require.NotContains(t, string(raw), "ana@correo.com")
	})

	t.Run("convierte el atributo error en la excepción del evento con su stacktrace", func(t *testing.T) {
		r, transport := newReporter(t)
		log := slog.New(r.Handler(slog.NewJSONHandler(&bytes.Buffer{}, nil)))

		log.Error("unhandled error", slog.Any("error", errors.New("inventory: insufficient stock")))

		require.Len(t, transport.sent(), 1)
		e := transport.sent()[0]
		require.Len(t, e.Exception, 1)
		require.Equal(t, "*errors.errorString", e.Exception[0].Type)
		require.Equal(t, "inventory: insufficient stock", e.Exception[0].Value)
		require.NotNil(t, e.Exception[0].Stacktrace)
		require.NotEmpty(t, e.Exception[0].Stacktrace.Frames)
	})

	t.Run("reporta un panic recuperado como excepción", func(t *testing.T) {
		r, transport := newReporter(t)
		log := slog.New(r.Handler(slog.NewJSONHandler(&bytes.Buffer{}, nil)))

		func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered", slog.Any("panic", rec))
				}
			}()
			panic("nil map write")
		}()

		require.Len(t, transport.sent(), 1)
		e := transport.sent()[0]
		require.Len(t, e.Exception, 1)
		require.Equal(t, "panic", e.Exception[0].Type)
		require.Equal(t, "nil map write", e.Exception[0].Value)
	})

	t.Run("pasa el evento por el filtro de datos sensibles", func(t *testing.T) {
		r, transport := newReporter(t)
		log := slog.New(r.Handler(slog.NewJSONHandler(&bytes.Buffer{}, nil)))

		log.Error("unhandled error", slog.Any("error", errors.New("user ana@correo.com: password=hunter2")))

		require.Len(t, transport.sent(), 1)
		raw, err := json.Marshal(transport.sent()[0])
		require.NoError(t, err)
		require.NotContains(t, string(raw), "ana@correo.com")
		require.NotContains(t, string(raw), "hunter2")
	})

	t.Run("marca el span activo con el id del evento", func(t *testing.T) {
		r, transport := newReporter(t)
		log := slog.New(r.Handler(slog.NewJSONHandler(&bytes.Buffer{}, nil)))
		recorder := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
		ctx, span := tp.Tracer("test").Start(context.Background(), "op")

		log.ErrorContext(ctx, "unhandled error")
		span.End()

		require.Len(t, transport.sent(), 1)
		ended := recorder.Ended()
		require.Len(t, ended, 1)
		var eventID string
		for _, kv := range ended[0].Attributes() {
			if kv.Key == "sentry.event_id" {
				eventID = kv.Value.AsString()
			}
		}
		require.Equal(t, string(transport.sent()[0].EventID), eventID)
	})

	t.Run("con DSN vacío no reporta nada y deja el handler intacto", func(t *testing.T) {
		r, err := errreport.New(errreport.Options{Environment: "test"})
		require.NoError(t, err)
		next := slog.NewJSONHandler(&bytes.Buffer{}, nil)

		require.Same(t, next, r.Handler(next))
		require.True(t, r.Flush(time.Second))
	})
}
