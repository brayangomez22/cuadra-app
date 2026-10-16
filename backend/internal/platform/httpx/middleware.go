package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// RequestIDHeader is the header used to propagate the request ID.
const RequestIDHeader = "X-Request-ID"

const maxRequestIDLen = 64

type requestIDKey struct{}

// RequestIDFromContext returns the request ID stored in ctx, or "".
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// RequestID ensures every request has an ID in its context and response header.
// An incoming X-Request-ID is reused only if it is short and made of safe
// characters, so it cannot inject content into logs or headers.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if !validRequestID(id) {
			id = newRequestID()
		}
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for _, c := range id {
		isAlnum := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !isAlnum && c != '-' && c != '_' && c != '.' {
			return false
		}
	}
	return true
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error.
	return hex.EncodeToString(b)
}

// Recover turns a panic into a 500 response with the standard error format.
// The panic value is logged but never sent to the client.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				// http.ErrAbortHandler is net/http's way to abort a response silently.
				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}
				logger.ErrorContext(r.Context(), "panic recovered",
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
				)
				WriteError(w, http.StatusInternalServerError, "internal_error", InternalErrorMessage)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Logging logs every request once it finishes. It logs only the path, never
// the query string, which may carry sensitive data.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			reqLog := &requestLog{}

			next.ServeHTTP(rec, r.WithContext(context.WithValue(r.Context(), requestLogKey{}, reqLog)))

			attrs := append([]slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			}, reqLog.attrs...)
			logger.LogAttrs(r.Context(), slog.LevelInfo, "http request", attrs...)
		})
	}
}

// requestLogKey is the context key of the request's *requestLog.
type requestLogKey struct{}

// requestLog holds the attributes that inner middleware add to the request's
// log line. Logging owns it; the handlers run on the same goroutine.
type requestLog struct{ attrs []slog.Attr }

// AddRequestLogAttrs adds attrs to the "http request" line that Logging writes
// when the request finishes. Logging wraps the middleware that learn who the
// caller is (auth), so their context never reaches it: this is how the tenant
// and the user get to that line. Outside Logging it does nothing.
func AddRequestLogAttrs(ctx context.Context, attrs ...slog.Attr) {
	if l, ok := ctx.Value(requestLogKey{}).(*requestLog); ok {
		l.attrs = append(l.attrs, attrs...)
	}
}

// statusRecorder captures the status code written by the next handler.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.wroteHeader = true
	return r.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// RequestIDAttrs returns the request ID as a log attribute. It is meant to be
// registered as a logger.ContextExtractor so every log line carries it.
func RequestIDAttrs(ctx context.Context) []slog.Attr {
	if id := RequestIDFromContext(ctx); id != "" {
		return []slog.Attr{slog.String("request_id", id)}
	}
	return nil
}
