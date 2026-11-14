// Package app holds the invoicing use cases: the tenant's settings and
// numbering resolutions, the document each confirmed sale gets (the sales
// module registers it through internal/server, in the sale's transaction),
// its issue by the technology provider in a background job, the periodic
// look up of the documents the DIAN has not answered and the manual retry.
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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing"

// latencyBuckets are the bucket boundaries of
// cuadra.invoicing.acceptance_latency, in seconds: from the provider's
// immediate answer to the DIAN's three hours of contingency.
var latencyBuckets = []float64{1, 2, 5, 10, 30, 60, 300, 900, 3600, 10800}

// Actor is who runs a use case. Both ids come from the authenticated
// principal, never from the request.
type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

// Deps are the Service's dependencies. Now defaults to time.Now, the
// providers to no-ops and Logger to a discarding logger.
type Deps struct {
	Settings    domain.SettingsRepository
	Resolutions domain.ResolutionRepository
	Documents   domain.DocumentRepository
	Invoicer    ElectronicInvoicer
	Sales       SaleSource
	Buyers      BuyerDirectory

	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Service runs the invoicing use cases.
type Service struct {
	Deps
	spans usecase.Spans

	documents      metric.Int64Counter
	latency        metric.Float64Histogram
	providerErrors metric.Int64Counter
	nearEnd        metric.Int64Counter
	retries        metric.Int64Counter
}

// NewService builds a Service. It fails if a dependency is missing or a
// metric instrument cannot be created.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.Settings, d.Resolutions, d.Documents, d.Invoicer, d.Sales, d.Buyers) {
		return nil, errors.New("invoicing: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)

	s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "invoicing")}
	err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
		usecase.Counter(&s.documents, "cuadra.invoicing.documents",
			"Electronic documents reaching a state, by document.kind and document.status.", "{document}"),
		usecase.Histogram(&s.latency, "cuadra.invoicing.acceptance_latency",
			"Time from a document's issue to its validation by the DIAN, by document.kind.", "s", latencyBuckets...),
		usecase.Counter(&s.providerErrors, "cuadra.invoicing.provider_errors",
			"Failures talking to the technology provider, by provider.reason.", "{error}"),
		usecase.Counter(&s.nearEnd, "cuadra.invoicing.numbering_near_end",
			"Numbering resolutions that crossed into their last tenth or their last 30 days, by document.kind.", "{resolution}"),
		usecase.Counter(&s.retries, "cuadra.invoicing.manual_retries",
			"Documents sent again by hand, by document.status before the retry.", "{document}"),
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

func documentAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("document.id", id.String())
}

func saleAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("sale.id", id.String())
}
