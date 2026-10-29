package app_test

import (
	"context"
	"slices"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

// fakeCatalog is an in-memory VariantCatalog.
type fakeCatalog struct {
	variants map[uuid.UUID]app.VariantInfo
}

func (f *fakeCatalog) Variants(_ context.Context, _ uuid.UUID, ids []uuid.UUID) ([]app.VariantInfo, error) {
	var out []app.VariantInfo
	for id, v := range f.variants {
		if slices.Contains(ids, id) {
			out = append(out, v)
		}
	}
	return out, nil
}

// fakeStock is a PurchaseStock that converts like the inventory does and
// records what it received. err, if set, fails the receipt.
type fakeStock struct {
	calls []stockCall
	err   error
}

type stockCall struct {
	locationID, purchaseID uuid.UUID
	lines                  []app.StockLine
}

func (f *fakeStock) ReceivePurchase(_ context.Context, _, _, locationID, purchaseID uuid.UUID, lines []app.StockLine) ([]domain.Received, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.calls = append(f.calls, stockCall{locationID: locationID, purchaseID: purchaseID, lines: lines})
	out := make([]domain.Received, len(lines))
	for i, l := range lines {
		out[i] = domain.Received{BaseQuantity: l.Quantity.Mul(l.Factor), BaseUnitCost: l.UnitCost.DivRound(l.Factor, 4)}
	}
	return out, nil
}
