// Package app holds the inventory use cases: locations, stock movements
// (receipts, adjustments, transfers) and the queries of stock and kardex.
package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory"

// Actor is who runs a use case. Both ids come from the authenticated
// principal, never from the request.
type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

// Deps are the Service's dependencies. Now defaults to time.Now, the
// providers to no-ops and Logger to a discarding logger.
type Deps struct {
	Locations      domain.LocationRepository
	Stock          domain.StockRepository
	Policies       domain.PolicyRepository
	Variants       ProductVariants
	Reorder        domain.ReorderRuleRepository
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// ProductVariants is what the inventory needs of the catalog, through a
// port internal/server satisfies with the catalog module: the product of each
// variant, for the product_id of the levels and movements.
type ProductVariants interface {
	// ProductsOf returns the product of each of variantIDs that exists in
	// tenantID; the others are left out.
	ProductsOf(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error)
	// UnitFactor returns how many base units one unitID of the variant
	// holds; domain.ErrUnitNotAvailable if the variant does not exist in
	// tenantID or has no such alternate unit.
	UnitFactor(ctx context.Context, tenantID, variantID, unitID uuid.UUID) (decimal.Decimal, error)
}

// Service runs the inventory use cases.
type Service struct {
	Deps
	spans usecase.Spans

	movements         metric.Int64Counter
	insufficientStock metric.Int64Counter
	reorderChanges    metric.Int64Counter
}

// NewService builds a Service. It fails if a repository is missing or a
// metric instrument cannot be created.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.Locations, d.Stock, d.Policies, d.Variants, d.Reorder) {
		return nil, errors.New("inventory: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)

	s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "inventory")}
	err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
		usecase.Counter(&s.movements, "cuadra.inventory.movements",
			"Stock movements registered, by type.", "{movement}"),
		usecase.Counter(&s.insufficientStock, "cuadra.inventory.insufficient_stock",
			"Stock exits rejected for lack of stock, by the movement type attempted.", "{rejection}"),
		usecase.Counter(&s.reorderChanges, "cuadra.inventory.reorder_rule_changes",
			"Reorder rules set or deleted, by action.", "{change}"),
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

func variantAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("variant.id", id.String())
}

func locationAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("location.id", id.String())
}
