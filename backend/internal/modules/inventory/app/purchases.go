package app

import (
	"bytes"
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// PurchaseStockLine is what a purchase line puts into the stock: Quantity
// of a unit holding Factor base units, at UnitCost per unit.
type PurchaseStockLine struct {
	VariantID uuid.UUID
	Quantity  decimal.Decimal
	UnitCost  decimal.Decimal
	Factor    decimal.Decimal
}

// PurchaseStockInput is the stock a purchase puts into a location.
type PurchaseStockInput struct {
	LocationID uuid.UUID
	PurchaseID uuid.UUID
	Lines      []PurchaseStockLine
}

// ReceivePurchaseStock registers the entries of a purchase: one
// purchase_in per line at the location, referencing the purchase, converted
// to the base unit with ToBaseUnit and the factor the purchase line keeps,
// and updating the average cost. The entries go in variant order, so two
// purchases (or a purchase and a sale) lock their levels in the same order,
// and come back in line order. Called within the purchase's transaction
// (through ctx), a failed entry rolls back the whole purchase; its error
// names the variant.
func (s *Service) ReceivePurchaseStock(ctx context.Context, a Actor, in PurchaseStockInput) (_ []*domain.StockMovement, err error) {
	ctx, span := s.startSpan(ctx, "ReceivePurchaseStock", a, locationAttr(in.LocationID), purchaseAttr(in.PurchaseID))
	defer func() { usecase.End(span, err) }()

	if err := s.requireActiveLocation(ctx, a, in.LocationID); err != nil {
		return nil, err
	}
	order := make([]int, len(in.Lines))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(x, y int) int {
		vx, vy := in.Lines[x].VariantID, in.Lines[y].VariantID
		return bytes.Compare(vx[:], vy[:])
	})

	info := s.info(a)
	info.Reference = &domain.DocumentRef{Kind: domain.DocumentPurchase, ID: in.PurchaseID}
	movements := make([]*domain.StockMovement, len(in.Lines))
	for _, i := range order {
		l := in.Lines[i]
		if movements[i], err = s.receivePurchaseLine(ctx, a, in.LocationID, l, info); err != nil {
			return nil, &VariantStockError{VariantID: l.VariantID, Err: err}
		}
	}
	s.countMovements(ctx, movements...)
	return movements, nil
}

// receivePurchaseLine converts a line to the base unit and enters it.
func (s *Service) receivePurchaseLine(ctx context.Context, a Actor, locationID uuid.UUID, l PurchaseStockLine, info domain.MovementInfo) (*domain.StockMovement, error) {
	qty, cost, err := domain.ToBaseUnit(l.Quantity, l.UnitCost, l.Factor)
	if err != nil {
		return nil, err
	}
	var m *domain.StockMovement
	err = s.Stock.Apply(ctx, a.TenantID, l.VariantID, []uuid.UUID{locationID},
		func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
			var err error
			m, err = levels[0].Receive(qty, cost, info)
			return []*domain.StockMovement{m}, err
		})
	return m, err
}

func purchaseAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("purchase.id", id.String())
}
