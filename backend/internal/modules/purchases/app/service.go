// Package app holds the purchases use cases: the tenant's suppliers and the
// purchases registered from their invoices, whose lines enter the stock
// through the inventory (a port that internal/server satisfies) in the same
// transaction.
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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

const instrumentationName = "github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases"

// ErrInvalidStatus reports a search by an unknown status.
var ErrInvalidStatus = errors.New("purchases: invalid status filter")

// Actor is who runs a use case. Both ids come from the authenticated
// principal, never from the request.
type Actor struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

// Deps are the Service's dependencies. Now defaults to time.Now, the
// providers to no-ops and Logger to a discarding logger.
type Deps struct {
	Suppliers      domain.SupplierRepository
	Purchases      domain.PurchaseRepository
	Catalog        VariantCatalog
	Stock          PurchaseStock
	Now            func() time.Time
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Logger         *slog.Logger
}

// Service runs the purchases use cases.
type Service struct {
	Deps
	spans usecase.Spans

	registered        metric.Int64Counter
	lines             metric.Int64Counter
	duplicateInvoices metric.Int64Counter
	suppliersCreated  metric.Int64Counter
}

// NewService builds a Service. It fails if a dependency is missing or a
// metric instrument cannot be created.
func NewService(d Deps) (*Service, error) {
	if usecase.AnyNil(d.Suppliers, d.Purchases, d.Catalog, d.Stock) {
		return nil, errors.New("purchases: missing service dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)

	s := &Service{Deps: d, spans: usecase.NewSpans(d.TracerProvider, instrumentationName, "purchases")}
	err := usecase.Create(usecase.Meter(d.MeterProvider, instrumentationName),
		usecase.Counter(&s.registered, "cuadra.purchases.registered",
			"Purchases registered from a supplier's invoice, by payment (cash or credit).", "{purchase}"),
		usecase.Counter(&s.lines, "cuadra.purchases.lines",
			"Lines of the registered purchases, by unit (base or alternate).", "{line}"),
		usecase.Counter(&s.duplicateInvoices, "cuadra.purchases.duplicate_invoices",
			"Purchases rejected because the supplier's invoice was already registered.", "{rejection}"),
		usecase.Counter(&s.suppliersCreated, "cuadra.purchases.suppliers_created",
			"Suppliers created.", "{supplier}"),
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

func supplierAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("supplier.id", id.String())
}

func purchaseAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("purchase.id", id.String())
}
