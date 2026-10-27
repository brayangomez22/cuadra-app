// Package app holds the catalog use cases: products, categories, units of
// measure, attribute definitions and variant axes of a tenant.
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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"

// Actor is who runs a use case. Both ids come from the authenticated
// principal, never from the request.
type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

// Deps are the Service's dependencies. Now defaults to time.Now, the
// providers to no-ops and Logger to a discarding logger.
type Deps struct {
	Products    domain.ProductRepository
	Categories  domain.CategoryRepository
	Units       domain.UnitRepository
	Attributes  domain.AttributeRepository
	VariantAxes domain.VariantAxisRepository
	Seeds       domain.SeedRepository
	// History tells whether variants have stock movements (the inventory's).
	History        StockHistory
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Service runs the catalog use cases.
type Service struct {
	Deps
	spans usecase.Spans

	productsCreated metric.Int64Counter
	variantsCreated metric.Int64Counter
	priceChanges    metric.Int64Counter
	searches        metric.Int64Counter
}

// NewService builds a Service. It fails if a repository is missing or a
// metric instrument cannot be created.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.Products, d.Categories, d.Units, d.Attributes, d.VariantAxes, d.Seeds, d.History) {
		return nil, errors.New("catalog: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)

	s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "catalog")}
	err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
		usecase.Counter(&s.productsCreated, "cuadra.catalog.products_created",
			"Products added to the catalog.", "{product}"),
		usecase.Counter(&s.variantsCreated, "cuadra.catalog.variants_created",
			"Variants added to an existing product (a product's default variant is not counted).", "{variant}"),
		usecase.Counter(&s.priceChanges, "cuadra.catalog.price_changes",
			"Variant price changes.", "{change}"),
		usecase.Counter(&s.searches, "cuadra.catalog.searches",
			"Product searches with a text, by result (hit, empty).", "{search}"),
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

func productAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("product.id", id.String())
}

func variantAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("variant.id", id.String())
}

func unitAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("unit.id", id.String())
}

func categoryAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("category.id", id.String())
}

func attributeAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("attribute.id", id.String())
}

func axisAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("axis.id", id.String())
}
