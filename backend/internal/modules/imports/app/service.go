// Package app holds the imports use cases: upload and validate a file,
// confirm it, write it by batches in a background job, and read its report
// and the template.
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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/imports"

// variantsModule is the ID of the activable module that lets a product
// take several rows, one per variant.
const variantsModule = "variants"

// Actor is who runs a use case. Both ids come from the authenticated
// principal or, in the job, from the import (who uploaded it).
type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

// Deps are the Service's dependencies. BatchRows defaults to
// domain.BatchRows, Now to time.Now, the providers to no-ops and Logger to
// a discarding logger.
type Deps struct {
	Imports        domain.ImportRepository
	Tx             Transactor
	Catalog        Catalog
	Stock          Stock
	Modules        Modules
	Sheets         Sheets
	BatchRows      int
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Service runs the imports use cases.
type Service struct {
	Deps
	spans usecase.Spans

	uploads  metric.Int64Counter
	runs     metric.Int64Counter
	products metric.Int64Counter
	duration metric.Float64Histogram
}

// NewService builds a Service. It fails if a dependency is missing or a
// metric instrument cannot be created.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.Imports, d.Tx, d.Catalog, d.Stock, d.Modules, d.Sheets) {
		return nil, errors.New("imports: missing service dependency")
	}
	if d.BatchRows <= 0 {
		d.BatchRows = domain.BatchRows
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)

	s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "imports")}
	err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
		usecase.Counter(&s.uploads, "cuadra.imports.uploads",
			"Import files uploaded, by result (validated, rejected).", "{upload}"),
		usecase.Counter(&s.runs, "cuadra.imports.runs",
			"Imports the job finished, by result (completed, failed).", "{run}"),
		usecase.Counter(&s.products, "cuadra.imports.products",
			"Products written by imports, by result (created, updated, skipped).", "{product}"),
		usecase.Histogram(&s.duration, "cuadra.imports.duration",
			"Time from the start of an import's job to its end.", "s",
			1, 5, 15, 30, 60, 120, 300, 600, 1800),
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// resultAttr is the result attribute of the module's counters.
func resultAttr(result string) metric.AddOption {
	return metric.WithAttributes(attribute.String("result", result))
}

// startSpan opens the use case's span with the actor's ids.
func (s *Service) startSpan(ctx context.Context, name string, a Actor, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return s.spans.StartFor(ctx, name, a.TenantID, a.UserID, attrs...)
}

func importAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("import.id", id.String())
}
