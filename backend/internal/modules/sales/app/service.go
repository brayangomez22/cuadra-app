// Package app holds the sales use cases: ConfirmSale (the point-of-sale
// checkout), VoidSale and the queries of a sale and of the day's sales. The
// catalog and the stock are reached through the ports
// ProductCatalog and StockReserver, which cmd/api satisfies with the other
// modules.
package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales"

// Actor is who runs a use case. The ids come from the authenticated
// principal, never from the request. UnlimitedDiscount is whether their role
// has the sales:discount permission; the HTTP adapter resolves it.
type Actor struct {
	TenantID          uuid.UUID
	UserID            uuid.UUID
	UnlimitedDiscount bool
}

// ProductInfo is what a sale needs of a product: its name, price without
// IVA, IVA rate and whether it can be sold.
type ProductInfo struct {
	ID      uuid.UUID
	Name    string
	Price   decimal.Decimal
	TaxRate decimal.Decimal
	Active  bool
}

// ProductCatalog reads products of the catalog module.
type ProductCatalog interface {
	// Products returns the products of tenantID among ids (distinct); the
	// ones that do not exist in the tenant are left out.
	Products(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]ProductInfo, error)
}

// StockLine is the quantity of a product a sale takes from stock.
type StockLine struct {
	ProductID uuid.UUID
	Quantity  decimal.Decimal
}

// StockReserver registers the stock exits of a sale in the inventory module.
type StockReserver interface {
	// ReserveSale registers one sale_out per line (one line per product) at
	// locationID, referencing saleID. It fails with a
	// *domain.InsufficientStockError (without the product's name) if the
	// tenant's stock policy rejects an exit, and with
	// domain.ErrLocationNotFound or domain.ErrLocationInactive if the
	// location cannot sell. It runs within the transaction of
	// SaleRepository.SaveConfirmed, through ctx.
	ReserveSale(ctx context.Context, tenantID, userID, locationID, saleID uuid.UUID, lines []StockLine) error
	// ReverseSale gives back the stock that saleID took, exactly, with
	// inverse movements. It runs within the transaction of
	// SaleRepository.Void, through ctx.
	ReverseSale(ctx context.Context, tenantID, userID, saleID uuid.UUID) error
}

// Deps are the Service's dependencies. Now defaults to time.Now, the
// providers to no-ops and Logger to a discarding logger.
type Deps struct {
	Products       ProductCatalog
	Stock          StockReserver
	Sales          domain.SaleRepository
	Policies       domain.PolicyRepository
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Service runs the sales use cases.
type Service struct {
	Deps
	tracer trace.Tracer

	confirmed        metric.Int64Counter
	amount           metric.Float64Histogram
	discountRejected metric.Int64Counter
	voided           metric.Int64Counter
}

// amountBuckets are the bucket boundaries of cuadra.sales.amount, in COP: a
// hardware store's sales go from a few screws to a load of cement.
var amountBuckets = []float64{
	5_000, 10_000, 25_000, 50_000, 100_000, 250_000, 500_000, 1_000_000, 2_500_000, 5_000_000, 10_000_000,
}

// NewService builds a Service. It fails if a port is missing or a metric
// instrument cannot be created.
func NewService(d Deps) (*Service, error) {
	if d.Products == nil || d.Stock == nil || d.Sales == nil || d.Policies == nil {
		return nil, errors.New("sales: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.TracerProvider == nil {
		d.TracerProvider = tracenoop.NewTracerProvider()
	}
	if d.MeterProvider == nil {
		d.MeterProvider = metricnoop.NewMeterProvider()
	}
	if d.Logger == nil {
		d.Logger = slog.New(slog.DiscardHandler)
	}

	s := &Service{Deps: d, tracer: d.TracerProvider.Tracer(instrumentationName)}
	meter := d.MeterProvider.Meter(instrumentationName)
	var err error
	if s.confirmed, err = meter.Int64Counter("cuadra.sales.confirmed",
		metric.WithDescription("Sales confirmed."),
		metric.WithUnit("{sale}")); err != nil {
		return nil, err
	}
	if s.amount, err = meter.Float64Histogram("cuadra.sales.amount",
		metric.WithDescription("Total of the sales confirmed, IVA included."),
		metric.WithUnit("{COP}"),
		metric.WithExplicitBucketBoundaries(amountBuckets...)); err != nil {
		return nil, err
	}
	if s.discountRejected, err = meter.Int64Counter("cuadra.sales.discount_rejected",
		metric.WithDescription("Sales rejected for a discount above the user's limit."),
		metric.WithUnit("{rejection}")); err != nil {
		return nil, err
	}
	if s.voided, err = meter.Int64Counter("cuadra.sales.voided",
		metric.WithDescription("Sales voided."),
		metric.WithUnit("{sale}")); err != nil {
		return nil, err
	}
	return s, nil
}

// startSpan opens the use case's span with the actor's ids.
func (s *Service) startSpan(ctx context.Context, name string, a Actor, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	attrs = append(attrs,
		attribute.String("tenant.id", a.TenantID.String()),
		attribute.String("user.id", a.UserID.String()),
	)
	return s.tracer.Start(ctx, "sales."+name, trace.WithAttributes(attrs...))
}

func saleAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("sale.id", id.String())
}

// endSpan records err on the span and ends it.
func endSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}
