// Package app holds the sales use cases: ConfirmSale (the point-of-sale
// checkout), VoidSale, the queries of a sale and of the day's sales, and the
// cash sessions (opening, cash ins and outs, closing with the count). The
// catalog and the stock are reached through the ports VariantCatalog and
// StockReserver, which internal/server satisfies with the other modules; prices come
// from a PriceResolver, a chain of rules that modules extend.
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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
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

// VariantInfo is what a sale needs of a variant: its product, the
// product's name, the variant's label ("" without axes), its price without
// IVA, the product's IVA rate and whether it can be sold (the variant and
// its product are active).
type VariantInfo struct {
	ID          uuid.UUID
	ProductID   uuid.UUID
	ProductName string
	Label       string
	Price       decimal.Decimal
	TaxRate     decimal.Decimal
	Active      bool
	// Units are the variant's alternate units.
	Units []UnitInfo
}

// UnitInfo is an alternate unit of a variant: a unit of the tenant, its
// code, how many base units it holds and its price without IVA.
type UnitInfo struct {
	UnitID uuid.UUID
	Code   string
	Factor decimal.Decimal
	Price  decimal.Decimal
}

// VariantCatalog reads variants of the catalog module.
type VariantCatalog interface {
	// Variants returns the variants of tenantID among ids (distinct); the
	// ones that do not exist in the tenant are left out.
	Variants(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]VariantInfo, error)
}

// StockLine is the quantity of a variant a sale takes from stock.
type StockLine struct {
	VariantID uuid.UUID
	Quantity  decimal.Decimal
}

// StockReserver registers the stock exits of a sale in the inventory module.
type StockReserver interface {
	// ReserveSale registers one sale_out per line (one line per variant) at
	// locationID, referencing saleID. It fails with a
	// *domain.InsufficientStockError (with only the variant's id) if the
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

// Deps are the Service's dependencies. Prices defaults to the PriceChain
// without rules (each variant at its price), Now to time.Now, the providers
// to no-ops and Logger to a discarding logger.
type Deps struct {
	Catalog        VariantCatalog
	Prices         PriceResolver
	Stock          StockReserver
	Sales          domain.SaleRepository
	Policies       domain.PolicyRepository
	CashSessions   domain.CashSessionRepository
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Service runs the sales use cases.
type Service struct {
	Deps
	spans usecase.Spans

	confirmed          metric.Int64Counter
	amount             metric.Float64Histogram
	discountRejected   metric.Int64Counter
	voided             metric.Int64Counter
	priceRules         metric.Int64Counter
	alternateUnitLines metric.Int64Counter

	cashOpened          metric.Int64Counter
	cashClosed          metric.Int64Counter
	cashDifference      metric.Float64Histogram
	cashMovements       metric.Int64Counter
	cashSessionRequired metric.Int64Counter
}

// amountBuckets are the bucket boundaries of cuadra.sales.amount, in COP: a
// hardware store's sales go from a few screws to a load of cement.
var amountBuckets = []float64{
	5_000, 10_000, 25_000, 50_000, 100_000, 250_000, 500_000, 1_000_000, 2_500_000, 5_000_000, 10_000_000,
}

// cashDifferenceBuckets are the bucket boundaries of
// cuadra.sales.cash_difference, in COP: from a missing coin to a missing
// bundle of bills.
var cashDifferenceBuckets = []float64{0, 50, 500, 1_000, 5_000, 10_000, 20_000, 50_000, 100_000, 500_000}

// NewService builds a Service. It fails if a port is missing or a metric
// instrument cannot be created.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.Catalog, d.Stock, d.Sales, d.Policies, d.CashSessions) {
		return nil, errors.New("sales: missing service dependency")
	}
	if d.Prices == nil {
		chain, err := NewPriceChain()
		if err != nil {
			return nil, err
		}
		d.Prices = chain
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)

	s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "sales")}
	if err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName), s.instruments()...); err != nil {
		return nil, err
	}
	return s, nil
}

// instruments are the metrics of the sales and of the cash sessions.
func (s *Service) instruments() []usecase.Instrument {
	return []usecase.Instrument{
		usecase.Counter(&s.confirmed, "cuadra.sales.confirmed", "Sales confirmed.", "{sale}"),
		usecase.Histogram(&s.amount, "cuadra.sales.amount",
			"Total of the sales confirmed, IVA included.", "{COP}", amountBuckets...),
		usecase.Counter(&s.discountRejected, "cuadra.sales.discount_rejected",
			"Sales rejected for a discount above the user's limit.", "{rejection}"),
		usecase.Counter(&s.voided, "cuadra.sales.voided", "Sales voided.", "{sale}"),
		usecase.Counter(&s.priceRules, "cuadra.sales.price_rule_applied",
			"Lines of confirmed sales priced, by the rule that gave the price (price.rule).", "{line}"),
		usecase.Counter(&s.alternateUnitLines, "cuadra.sales.alternate_unit_lines",
			"Lines of confirmed sales sold in an alternate unit of the variant.", "{line}"),

		usecase.Counter(&s.cashOpened, "cuadra.sales.cash_sessions_opened", "Cash sessions opened.", "{session}"),
		usecase.Counter(&s.cashClosed, "cuadra.sales.cash_sessions_closed",
			"Cash sessions closed, by result of the count (cash.result).", "{session}"),
		usecase.Histogram(&s.cashDifference, "cuadra.sales.cash_difference",
			"Absolute difference between the counted and the expected cash at closing, by cash.result.", "{COP}",
			cashDifferenceBuckets...),
		usecase.Counter(&s.cashMovements, "cuadra.sales.cash_movements",
			"Cash ins and outs recorded, by cash.movement_kind.", "{movement}"),
		usecase.Counter(&s.cashSessionRequired, "cuadra.sales.cash_session_required_rejected",
			"Sales rejected because the cashier had no open cash session.", "{rejection}"),
	}
}

// startSpan opens the use case's span with the actor's ids.
func (s *Service) startSpan(ctx context.Context, name string, a Actor, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return s.spans.StartFor(ctx, name, a.TenantID, a.UserID, attrs...)
}

func saleAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("sale.id", id.String())
}

func cashSessionAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("cash_session.id", id.String())
}
