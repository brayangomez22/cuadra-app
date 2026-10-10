package httpx_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// errOutOfStock plays the role of a domain error such as inventory's ErrInsufficientStock.
var errOutOfStock = errors.New("insufficient stock")

func TestErrorMapperResponseError(t *testing.T) {
	rule := httpx.ErrorRule{
		Err:     errOutOfStock,
		Status:  http.StatusUnprocessableEntity,
		Code:    "insufficient_stock",
		Message: "No hay stock suficiente.",
	}

	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{
			name:        "un error de dominio mapeado devuelve su status, code y mensaje",
			err:         errOutOfStock,
			wantStatus:  http.StatusUnprocessableEntity,
			wantCode:    "insufficient_stock",
			wantMessage: "No hay stock suficiente.",
		},
		{
			name:        "un error de dominio envuelto con %w también se mapea",
			err:         fmt.Errorf("confirm sale: %w", errOutOfStock),
			wantStatus:  http.StatusUnprocessableEntity,
			wantCode:    "insufficient_stock",
			wantMessage: "No hay stock suficiente.",
		},
		{
			name:        "un error no mapeado devuelve 500 internal_error",
			err:         errors.New("connection reset by peer"),
			wantStatus:  http.StatusInternalServerError,
			wantCode:    "internal_error",
			wantMessage: "Ocurrió un error inesperado. Intenta de nuevo.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := httpx.NewErrorMapper(discardLogger(), rule)
			rec := httptest.NewRecorder()

			mapper.ResponseError(rec, httptest.NewRequest(http.MethodPost, "/api/v1/sales", nil), tt.err)

			require.Equal(t, tt.wantStatus, rec.Code)
			requireErrorBody(t, rec, tt.wantCode)
			var body httpx.ErrorBody
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, tt.wantMessage, body.Error.Message)
		})
	}

	t.Run("un error no mapeado no filtra el mensaje interno y queda en el log y en el span", func(t *testing.T) {
		recorder := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
		t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })
		ctx, span := tp.Tracer("test").Start(t.Context(), "request")
		var logs bytes.Buffer
		mapper := httpx.NewErrorMapper(slog.New(slog.NewJSONHandler(&logs, nil)), rule)
		rec := httptest.NewRecorder()

		mapper.ResponseError(rec, httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/sales", nil), errors.New("connection reset by peer"))
		span.End()

		require.NotContains(t, rec.Body.String(), "connection reset")
		var entry map[string]any
		require.NoError(t, json.Unmarshal(logs.Bytes(), &entry))
		require.Equal(t, "ERROR", entry["level"])
		require.Contains(t, entry["error"], "connection reset by peer")
		spans := recorder.Ended()
		require.Len(t, spans, 1)
		require.Equal(t, codes.Error, spans[0].Status().Code)
		require.NotEmpty(t, spans[0].Events(), "el error queda registrado como evento del span")
	})
}

func TestErrorMapperRequestError(t *testing.T) {
	t.Run("un error al decodificar la request devuelve 400 invalid_request", func(t *testing.T) {
		mapper := httpx.NewErrorMapper(discardLogger())
		rec := httptest.NewRecorder()

		mapper.RequestError(rec, httptest.NewRequest(http.MethodPost, "/api/v1/products", nil), errors.New("can't decode JSON body: unexpected EOF"))

		require.Equal(t, http.StatusBadRequest, rec.Code)
		requireErrorBody(t, rec, "invalid_request")
		require.NotContains(t, rec.Body.String(), "unexpected EOF", "no expone detalles internos")
	})
}

// requireErrorBody checks that rec holds the standard Error schema with code.
func requireErrorBody(t *testing.T, rec *httptest.ResponseRecorder, code string) {
	t.Helper()
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var body httpx.ErrorBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "body: %s", rec.Body.String())
	require.Equal(t, code, body.Error.Code)
	require.NotEmpty(t, body.Error.Message)
}
