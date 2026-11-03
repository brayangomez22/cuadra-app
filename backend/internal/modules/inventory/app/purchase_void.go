package app

import (
	"bytes"
	"context"
	"slices"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ReversePurchaseStock takes out the stock a voided purchase put in: for each
// of its purchase_in movements, a purchase_void of the same quantity at the
// same location and cost. It fails with a VariantStockError wrapping
// ErrInsufficientStock if a variant no longer has that stock, whatever the
// tenant's policy, and with ErrPurchaseAlreadyReversed if the purchase's
// stock was already taken out. The location may be inactive. Called within
// the purchase's transaction (through ctx), a failed exit leaves it as it was.
func (s *Service) ReversePurchaseStock(ctx context.Context, a Actor, purchaseID uuid.UUID) (_ []*domain.StockMovement, err error) {
	ctx, span := s.startSpan(ctx, "ReversePurchaseStock", a, purchaseAttr(purchaseID))
	defer func() { usecase.End(span, err) }()

	ref := domain.DocumentRef{Kind: domain.DocumentPurchase, ID: purchaseID}
	existing, err := s.Stock.MovementsByReference(ctx, a.TenantID, ref)
	if err != nil {
		return nil, err
	}
	var entries []*domain.StockMovement
	for _, m := range existing {
		switch m.Type() {
		case domain.MovementPurchaseVoid:
			return nil, domain.ErrPurchaseAlreadyReversed
		case domain.MovementPurchaseIn:
			entries = append(entries, m)
		}
	}
	sortByLevel(entries) // same lock order as the purchase's entries and the sales

	info := s.info(a)
	info.Reference = &ref
	movements := make([]*domain.StockMovement, 0, len(entries))
	for _, entry := range entries {
		var m *domain.StockMovement
		err := s.Stock.Apply(ctx, a.TenantID, entry.VariantID(), []uuid.UUID{entry.LocationID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				var err error
				m, err = levels[0].ReturnPurchase(entry.Quantity(), entry.UnitCost(), info)
				return []*domain.StockMovement{m}, err
			})
		if err != nil {
			s.countInsufficient(ctx, err, domain.MovementPurchaseVoid)
			return nil, &VariantStockError{VariantID: entry.VariantID(), Err: err}
		}
		movements = append(movements, m)
	}
	s.countMovements(ctx, movements...)
	return movements, nil
}

// sortByLevel orders movements by variant and then location, the order the
// levels are locked in, so two operations on the same variants cannot
// deadlock.
func sortByLevel(movements []*domain.StockMovement) {
	slices.SortFunc(movements, func(x, y *domain.StockMovement) int {
		vx, vy := x.VariantID(), y.VariantID()
		if c := bytes.Compare(vx[:], vy[:]); c != 0 {
			return c
		}
		lx, ly := x.LocationID(), y.LocationID()
		return bytes.Compare(lx[:], ly[:])
	})
}
