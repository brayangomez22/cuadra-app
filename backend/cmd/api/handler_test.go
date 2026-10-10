package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/logger"
)

func TestHandlerTelemetry(t *testing.T) {
	setup := func(t *testing.T) (http.Handler, *tracetest.SpanRecorder, *bytes.Buffer) {
		t.Helper()
		recorder := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
		t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })
		var buf bytes.Buffer
		log := slog.New(logger.NewContextHandler(slog.NewJSONHandler(&buf, nil), httpx.RequestIDAttrs))
		return newHandler(log, tp), recorder, &buf
	}

	t.Run("una request a /healthz produce un span con la ruta y el status", func(t *testing.T) {
		handler, recorder, _ := setup(t)

		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

		spans := recorder.Ended()
		require.Len(t, spans, 1)
		require.Equal(t, "GET /healthz", spans[0].Name())
		attrs := map[attribute.Key]attribute.Value{}
		for _, kv := range spans[0].Attributes() {
			attrs[kv.Key] = kv.Value
		}
		require.Equal(t, "/healthz", attrs["http.route"].AsString())
		require.EqualValues(t, http.StatusOK, attrs["http.response.status_code"].AsInt64())
	})

	t.Run("los logs de una request llevan el trace_id de su span", func(t *testing.T) {
		handler, recorder, buf := setup(t)
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set(httpx.RequestIDHeader, "req-42")

		handler.ServeHTTP(httptest.NewRecorder(), req)

		spans := recorder.Ended()
		require.Len(t, spans, 1)
		var entry map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
		require.Equal(t, "http request", entry["msg"])
		require.Equal(t, spans[0].SpanContext().TraceID().String(), entry["trace_id"])
		require.Equal(t, "req-42", entry["request_id"])
	})
}
