// Package app holds the reports' use cases: each checks its filters, reads
// its report through a reader port and hides the costs from whoever may
// not see them. Nothing here writes.
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

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/reports"

// Actor is who asks for a report: ids and role from the access token, and
// whether the role sees costs and margins (auth.SeesCosts).
type Actor struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	Role      string
	SeesCosts bool
}

// Deps are the Service's dependencies.
type Deps struct {
	Sales          domain.SalesReader
	Stock          domain.StockReader
	Team           domain.TeamReader
	Receivables    Receivables
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Service runs the reports' use cases.
type Service struct {
	Deps
	spans    usecase.Spans
	queries  metric.Int64Counter
	duration metric.Float64Histogram
	exports  metric.Int64Counter
}

// NewService builds a Service. It fails if a reader is missing or a metric
// instrument cannot be created. Now defaults to time.Now, the providers to
// no-ops and Logger to a discarding logger.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.Sales, d.Stock, d.Team, d.Receivables) {
		return nil, errors.New("reports: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)
	s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "reports")}
	err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
		usecase.Counter(&s.queries, "cuadra.reports.queries",
			"Business reports read, by report.", "{report}"),
		usecase.Histogram(&s.duration, "cuadra.reports.duration",
			"Time to read a business report, by report.", "ms"),
		usecase.Counter(&s.exports, "cuadra.reports.exports",
			"Business reports exported to a file, by report and format.", "{file}"),
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// startSpan opens the use case's span with the actor's ids.
func (s *Service) startSpan(ctx context.Context, name string, a Actor) (context.Context, trace.Span) {
	return s.spans.StartFor(ctx, name, a.TenantID, a.UserID)
}

// observe counts a read of report and records how long it took since
// start.
func (s *Service) observe(ctx context.Context, report string, start time.Time) {
	attrs := metric.WithAttributes(attribute.String("report", report))
	s.queries.Add(ctx, 1, attrs)
	s.duration.Record(ctx, float64(time.Since(start).Microseconds())/1000, attrs)
}

// Exported counts a report exported to a file. It opens no span: the
// report's own use case traced the read.
func (s *Service) Exported(ctx context.Context, report, format string) {
	s.exports.Add(ctx, 1, metric.WithAttributes(attribute.String("report", report), attribute.String("format", format)))
}

// today is the current day in Colombia.
func (s *Service) today() calendar.Date { return calendar.DateIn(s.Now()) }

// scope is the actor's tenant and the optional location.
func scope(a Actor, locationID *uuid.UUID) domain.Scope {
	return domain.Scope{TenantID: a.TenantID, LocationID: locationID}
}

// Range are the days of a report, both included, in Colombia.
type Range struct{ From, To calendar.Date }

func (r Range) period() (domain.Period, error) { return domain.NewPeriod(r.From, r.To) }
