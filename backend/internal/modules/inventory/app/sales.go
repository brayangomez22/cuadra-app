package app

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// SaleStockLine is the quantity of a product a sale takes; a sale has one
// line per product.
type SaleStockLine struct {
	ProductID uuid.UUID
	Quantity  decimal.Decimal
}

// SaleStockInput is the stock a sale takes from a location.
type SaleStockInput struct {
	LocationID uuid.UUID
	SaleID     uuid.UUID
	Lines      []SaleStockLine
}

// ProductStockError is the error of one product's exit; it unwraps to the
// domain error (ErrInsufficientStock, ErrProductNotFound...), so the sales
// module can tell the cashier which product failed.
type ProductStockError struct {
	ProductID uuid.UUID
	Err       error
}

func (e *ProductStockError) Error() string {
	return fmt.Sprintf("product %s: %v", e.ProductID, e.Err)
}

func (e *ProductStockError) Unwrap() error { return e.Err }

// IssueSaleStock registers the exits of a sale: one sale_out per product at
// the location, referencing the sale and following the tenant's stock
// policy. The exits go in product order, so two sales of the same products
// lock their levels in the same order and cannot deadlock. Called within the
// sale's transaction (through ctx), a failed exit rolls back the whole sale.
func (s *Service) IssueSaleStock(ctx context.Context, a Actor, in SaleStockInput) (_ []*domain.StockMovement, err error) {
	ctx, span := s.startSpan(ctx, "IssueSaleStock", a, locationAttr(in.LocationID), saleAttr(in.SaleID))
	defer func() { endSpan(span, err) }()

	if err := s.requireActiveLocation(ctx, a, in.LocationID); err != nil {
		return nil, err
	}
	policy, err := s.Policies.Get(ctx, a.TenantID)
	if err != nil {
		return nil, err
	}
	lines := slices.Clone(in.Lines)
	slices.SortFunc(lines, func(x, y SaleStockLine) int { return bytes.Compare(x.ProductID[:], y.ProductID[:]) })

	info := s.info(a)
	info.Reference = &domain.DocumentRef{Kind: domain.DocumentSale, ID: in.SaleID}
	movements := make([]*domain.StockMovement, 0, len(lines))
	for _, l := range lines {
		var m *domain.StockMovement
		err := s.Stock.Apply(ctx, a.TenantID, l.ProductID, []uuid.UUID{in.LocationID},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				var err error
				m, err = levels[0].Issue(l.Quantity, policy, info)
				return []*domain.StockMovement{m}, err
			})
		if err != nil {
			s.countInsufficient(ctx, err, domain.MovementSaleOut)
			return nil, &ProductStockError{ProductID: l.ProductID, Err: err}
		}
		movements = append(movements, m)
	}
	s.countMovements(ctx, movements...)
	return movements, nil
}

// ReverseSaleStock gives back the stock a sale took: for each of its
// sale_out movements, a sale_void of the same quantity at the same location,
// valued at the cost the exit had. It fails with ErrSaleAlreadyReversed if
// the sale's stock was already given back. A sale without exits gives back
// nothing. The location may be inactive: the goods are already there.
func (s *Service) ReverseSaleStock(ctx context.Context, a Actor, saleID uuid.UUID) (_ []*domain.StockMovement, err error) {
	ctx, span := s.startSpan(ctx, "ReverseSaleStock", a, saleAttr(saleID))
	defer func() { endSpan(span, err) }()

	ref := domain.DocumentRef{Kind: domain.DocumentSale, ID: saleID}
	existing, err := s.Stock.MovementsByReference(ctx, a.TenantID, ref)
	if err != nil {
		return nil, err
	}
	var exits []*domain.StockMovement
	for _, m := range existing {
		switch m.Type() {
		case domain.MovementSaleVoid:
			return nil, domain.ErrSaleAlreadyReversed
		case domain.MovementSaleOut:
			exits = append(exits, m)
		}
	}
	// Same lock order as IssueSaleStock.
	slices.SortFunc(exits, func(x, y *domain.StockMovement) int {
		px, py := x.ProductID(), y.ProductID()
		if c := bytes.Compare(px[:], py[:]); c != 0 {
			return c
		}
		lx, ly := x.LocationID(), y.LocationID()
		return bytes.Compare(lx[:], ly[:])
	})

	info := s.info(a)
	info.Reference = &ref
	movements := make([]*domain.StockMovement, 0, len(exits))
	for _, exit := range exits {
		var m *domain.StockMovement
		err := s.Stock.Apply(ctx, a.TenantID, exit.ProductID(), []uuid.UUID{exit.LocationID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				var err error
				m, err = levels[0].ReturnSale(exit.Quantity().Neg(), exit.UnitCost(), info)
				return []*domain.StockMovement{m}, err
			})
		if err != nil {
			return nil, err
		}
		movements = append(movements, m)
	}
	s.countMovements(ctx, movements...)
	return movements, nil
}

func saleAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("sale.id", id.String())
}
