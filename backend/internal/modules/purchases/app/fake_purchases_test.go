package app_test

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

// fakePurchases is an in-memory PurchaseRepository. Like the real one, it
// checks the invoice before calling receive and stores nothing if receive
// fails.
type fakePurchases struct {
	rows []domain.PurchaseSnapshot
}

func (f *fakePurchases) Save(ctx context.Context, p *domain.Purchase, receive func(context.Context) error) error {
	for _, r := range f.rows {
		other, err := domain.RehydratePurchase(r)
		if err != nil {
			return err
		}
		if r.TenantID == p.TenantID() && r.SupplierID == p.SupplierID() && other.InvoiceKey() == p.InvoiceKey() {
			return domain.ErrInvoiceTaken
		}
	}
	if err := receive(ctx); err != nil {
		return err
	}
	f.rows = append(f.rows, p.Snapshot())
	return nil
}

func (f *fakePurchases) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Purchase, error) {
	for _, r := range f.rows {
		if r.ID == id && r.TenantID == tenantID {
			return domain.RehydratePurchase(r)
		}
	}
	return nil, domain.ErrPurchaseNotFound
}

func (f *fakePurchases) Search(_ context.Context, flt domain.PurchaseFilter) ([]domain.PurchaseSummary, int, error) {
	var out []domain.PurchaseSummary
	for _, r := range slices.Backward(f.rows) {
		p, err := domain.RehydratePurchase(r)
		if err != nil {
			return nil, 0, err
		}
		if r.TenantID != flt.TenantID || (flt.SupplierID != nil && *flt.SupplierID != r.SupplierID) ||
			(flt.Status != "" && flt.Status != p.Status()) {
			continue
		}
		out = append(out, domain.PurchaseSummary{
			ID: p.ID(), SupplierID: p.SupplierID(), InvoiceNumber: p.InvoiceNumber(), IssuedOn: p.IssuedOn(),
			Payment: p.Payment(), Total: p.Totals().Total, BalanceDue: p.BalanceDue(), Lines: len(p.Lines()),
		})
	}
	return out, len(out), nil
}

func (f *fakePurchases) BalancesDue(_ context.Context, tenantID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	out := map[uuid.UUID]decimal.Decimal{}
	for _, r := range f.rows {
		if r.TenantID == tenantID && slices.Contains(ids, r.SupplierID) && r.BalanceDue.IsPositive() {
			out[r.SupplierID] = out[r.SupplierID].Add(r.BalanceDue)
		}
	}
	return out, nil
}
