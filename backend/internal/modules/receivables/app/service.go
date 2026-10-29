// Package app holds the receivables use cases: the credit terms of a
// customer, the receivables that credit sales open (the sales module charges
// and cancels them through internal/server), the payments that settle them,
// the customer's statement and the aging of what the tenant's customers owe.
package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables"

// paymentAmountBuckets are the bucket boundaries of
// cuadra.receivables.payment_amount, in COP.
var paymentAmountBuckets = []float64{10_000, 50_000, 100_000, 250_000, 500_000, 1_000_000, 5_000_000}

// Actor is who runs a use case. Both ids come from the authenticated
// principal, never from the request.
type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

// Deps are the Service's dependencies. Now defaults to time.Now, the
// providers to no-ops and Logger to a discarding logger.
type Deps struct {
	Credit         domain.CreditRepository
	Customers      CustomerDirectory
	Cash           CashDrawer
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Service runs the receivables use cases.
type Service struct {
	Deps
	spans usecase.Spans

	charges         metric.Int64Counter
	limitRejections metric.Int64Counter
	cancellations   metric.Int64Counter
	payments        metric.Int64Counter
	paymentAmount   metric.Float64Histogram
}

// NewService builds a Service. It fails if a dependency is missing or a
// metric instrument cannot be created.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.Credit, d.Customers, d.Cash) {
		return nil, errors.New("receivables: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)

	s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "receivables")}
	err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
		usecase.Counter(&s.charges, "cuadra.receivables.charges",
			"Credit sales charged to a customer, by credit.authorized (whether they exceeded the limit).", "{sale}"),
		usecase.Counter(&s.limitRejections, "cuadra.receivables.limit_rejections",
			"Credit sales rejected because they exceeded the customer's available credit.", "{sale}"),
		usecase.Counter(&s.cancellations, "cuadra.receivables.cancellations",
			"Receivables canceled because their sale was voided.", "{sale}"),
		usecase.Counter(&s.payments, "cuadra.receivables.payments",
			"Payments received from customers, by payment.method.", "{payment}"),
		usecase.Histogram(&s.paymentAmount, "cuadra.receivables.payment_amount",
			"Amount of the payments received from customers.", "{COP}", paymentAmountBuckets...),
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// startSpan opens the use case's span with the actor's ids.
func (s *Service) startSpan(ctx context.Context, name string, a Actor, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return s.spans.StartFor(ctx, name, a.TenantID, a.UserID, attrs...)
}

func customerAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("customer.id", id.String())
}

func saleAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("sale.id", id.String())
}
