package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

func recordSpans(t *testing.T) (*tracetest.SpanRecorder, usecase.Spans) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	return recorder, usecase.NewSpans(tp, "test/instrumentation", "catalog")
}

func TestSpans(t *testing.T) {
	tenantID := uuid.MustParse("01920000-0000-7000-8000-000000000001")
	userID := uuid.MustParse("01920000-0000-7000-8000-000000000002")
	productID := attribute.String("product.id", "p-1")

	t.Run("nombra el span con el módulo y el caso de uso", func(t *testing.T) {
		recorder, spans := recordSpans(t)

		_, span := spans.Start(context.Background(), "CreateProduct", productID)
		usecase.End(span, nil)

		ended := recorder.Ended()
		require.Len(t, ended, 1)
		require.Equal(t, "catalog.CreateProduct", ended[0].Name())
		require.Equal(t, []attribute.KeyValue{productID}, ended[0].Attributes())
		require.Equal(t, "test/instrumentation", ended[0].InstrumentationScope().Name)
	})

	t.Run("StartFor agrega el tenant y el usuario después de los atributos del caso de uso", func(t *testing.T) {
		recorder, spans := recordSpans(t)

		_, span := spans.StartFor(context.Background(), "ConfirmSale", tenantID, userID, productID)
		usecase.End(span, nil)

		require.Equal(t, []attribute.KeyValue{
			productID,
			attribute.String("tenant.id", tenantID.String()),
			attribute.String("user.id", userID.String()),
		}, recorder.Ended()[0].Attributes())
	})

	t.Run("End registra el error en el span y lo marca como fallido", func(t *testing.T) {
		recorder, spans := recordSpans(t)
		failure := errors.New("no hay stock suficiente")

		_, span := spans.Start(context.Background(), "ReceiveGoods")
		usecase.End(span, failure)

		ended := recorder.Ended()[0]
		require.Equal(t, codes.Error, ended.Status().Code)
		require.Equal(t, failure.Error(), ended.Status().Description)
		require.Len(t, ended.Events(), 1)
		require.Equal(t, "exception", ended.Events()[0].Name)
	})

	t.Run("End sin error deja el estado sin marcar", func(t *testing.T) {
		recorder, spans := recordSpans(t)

		_, span := spans.Start(context.Background(), "ListUnits")
		usecase.End(span, nil)

		ended := recorder.Ended()[0]
		require.Equal(t, codes.Unset, ended.Status().Code)
		require.Empty(t, ended.Events())
	})

	t.Run("sin proveedor de trazas no falla (no-op)", func(t *testing.T) {
		spans := usecase.NewSpans(nil, "test/instrumentation", "sales")

		ctx, span := spans.Start(context.Background(), "VoidSale")
		usecase.End(span, errors.New("x"))

		require.NotNil(t, ctx)
		require.False(t, span.IsRecording())
	})
}
