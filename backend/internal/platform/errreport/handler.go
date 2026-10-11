package errreport

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// eventIDAttr marks the span that was active when the error was reported, so
// a trace in Tempo leads to its Sentry event.
const eventIDAttr = "sentry.event_id"

// tagKeys are the extractor attributes that become Sentry tags.
var tagKeys = map[string]bool{"tenant_id": true, "request_id": true}

// handler passes every record to next and, for records at level Error, also
// sends an event to Sentry.
type handler struct {
	next     slog.Handler
	reporter *Reporter
}

func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= slog.LevelError || h.next.Enabled(ctx, level)
}

func (h *handler) Handle(ctx context.Context, rec slog.Record) error {
	if rec.Level >= slog.LevelError {
		h.reporter.capture(ctx, rec)
	}
	if !h.next.Enabled(ctx, rec.Level) {
		return nil
	}
	return h.next.Handle(ctx, rec)
}

// WithAttrs keeps the attributes for the log output only: they never reach
// Sentry, so an "error" attribute must be on the record itself to become the
// event's exception.
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handler{next: h.next.WithAttrs(attrs), reporter: h.reporter}
}

func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{next: h.next.WithGroup(name), reporter: h.reporter}
}

// capture builds the event from the allowlisted parts of the record and sends it.
func (r *Reporter) capture(ctx context.Context, rec slog.Record) {
	event := sentry.NewEvent()
	event.Level = sentry.LevelError
	event.Message = rec.Message
	event.Timestamp = rec.Time

	if exc, ok := exceptionFrom(rec); ok {
		event.Exception = []sentry.Exception{exc}
	}

	for _, extract := range r.extractors {
		for _, a := range extract(ctx) {
			switch {
			case tagKeys[a.Key]:
				event.Tags[a.Key] = a.Value.String()
			case a.Key == "user_id":
				event.User.ID = a.Value.String()
			}
		}
	}

	span := trace.SpanFromContext(ctx)
	if sc := span.SpanContext(); sc.IsValid() {
		event.Tags["trace_id"] = sc.TraceID().String()
		event.Contexts["trace"] = sentry.Context{
			"trace_id": sc.TraceID().String(),
			"span_id":  sc.SpanID().String(),
		}
	}

	// No scope: the event carries only what was set above.
	if id := r.client.CaptureEvent(event, nil, nil); id != nil {
		span.SetAttributes(attribute.String(eventIDAttr, string(*id)))
	}
}

// exceptionFrom turns the record's "error" or "panic" attribute into a Sentry
// exception, with the stack trace of the goroutine that logged it. Logged from
// a deferred recover, that stack still holds the frames that panicked.
func exceptionFrom(rec slog.Record) (sentry.Exception, bool) {
	var exc sentry.Exception
	found := false
	rec.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "error":
			v := a.Value.Any()
			if err, ok := v.(error); ok {
				exc = sentry.Exception{Type: reflect.TypeOf(err).String(), Value: err.Error()}
			} else {
				exc = sentry.Exception{Type: "error", Value: fmt.Sprint(v)}
			}
		case "panic":
			exc = sentry.Exception{Type: "panic", Value: fmt.Sprint(a.Value.Any())}
		default:
			return true
		}
		found = true
		return false
	})
	if !found {
		return exc, false
	}
	exc.Stacktrace = callerStacktrace()
	return exc, true
}

// callerStacktrace is the current stack without the frames of the logging
// machinery (log/slog and this package), so Sentry groups events by the code
// that failed.
func callerStacktrace() *sentry.Stacktrace {
	st := sentry.NewStacktrace()
	if st == nil {
		return nil
	}
	frames := st.Frames[:0]
	for _, f := range st.Frames {
		if f.Module == "log/slog" || strings.HasSuffix(f.Module, "/platform/errreport") {
			continue
		}
		frames = append(frames, f)
	}
	st.Frames = frames
	return st
}
