package app_test

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
)

// store keeps one tenant's credit in memory, as the repository would store
// it. Update works on copies and keeps them only if fn succeeds, like a
// transaction; the drawer's cash ins are part of it.
type store struct {
	accounts    map[uuid.UUID]domain.AccountSnapshot
	receivables map[uuid.UUID]domain.ReceivableSnapshot
	payments    []domain.Payment
	cashIns     []decimal.Decimal
}

func newStore() *store {
	return &store{accounts: map[uuid.UUID]domain.AccountSnapshot{}, receivables: map[uuid.UUID]domain.ReceivableSnapshot{}}
}

type fakeCredit struct{ s *store }

func (f fakeCredit) Account(_ context.Context, tenantID, customerID uuid.UUID) (*domain.Account, error) {
	if snap, ok := f.s.accounts[customerID]; ok {
		return domain.RehydrateAccount(snap), nil
	}
	return domain.RehydrateAccount(domain.AccountSnapshot{TenantID: tenantID, CustomerID: customerID}), nil
}

func (f fakeCredit) SaveTerms(_ context.Context, a *domain.Account, _ time.Time) error {
	snap := f.s.accounts[a.CustomerID()]
	snap.TenantID, snap.CustomerID = a.TenantID(), a.CustomerID()
	snap.Limit, snap.TermDays = a.Terms().Limit(), a.Terms().Days()
	f.s.accounts[a.CustomerID()] = snap
	return nil
}

func (f fakeCredit) Update(ctx context.Context, tenantID, customerID uuid.UUID,
	fn func(context.Context, *domain.Account, []*domain.Receivable) (domain.Change, error),
) error {
	a, _ := f.Account(ctx, tenantID, customerID)
	open, _ := f.Open(ctx, tenantID, customerID)
	cashIns := len(f.s.cashIns)
	change, err := fn(ctx, a, open)
	if err != nil {
		f.s.cashIns = f.s.cashIns[:cashIns]
		return err
	}
	snap := f.s.accounts[customerID]
	snap.TenantID, snap.CustomerID, snap.Balance = tenantID, customerID, a.Balance()
	f.s.accounts[customerID] = snap
	for _, r := range append(change.Changed, change.Opened) {
		if r != nil {
			f.s.receivables[r.ID()] = r.Snapshot()
		}
	}
	if change.Payment != nil {
		f.s.payments = append(f.s.payments, *change.Payment)
	}
	return nil
}

func (f fakeCredit) OfSale(_ context.Context, _, saleID uuid.UUID) (domain.SaleReceivable, error) {
	for _, r := range f.s.receivables {
		if r.SaleID == saleID {
			return domain.SaleReceivable{ID: r.ID, CustomerID: r.CustomerID, Status: r.Status}, nil
		}
	}
	return domain.SaleReceivable{}, domain.ErrReceivableNotFound
}

func (f fakeCredit) Open(_ context.Context, _, customerID uuid.UUID) ([]*domain.Receivable, error) {
	var out []*domain.Receivable
	for _, r := range f.s.receivables {
		if r.CustomerID == customerID && r.Status == domain.ReceivableOpen {
			out = append(out, domain.RehydrateReceivable(r))
		}
	}
	slices.SortFunc(out, func(x, y *domain.Receivable) int { return x.DueDate().Time().Compare(y.DueDate().Time()) })
	return out, nil
}

// Movements lists the charges and the payments; the cancellations are the
// repository's (integration tests).
func (f fakeCredit) Movements(_ context.Context, _, customerID uuid.UUID) ([]domain.Movement, error) {
	var out []domain.Movement
	for _, r := range f.s.receivables {
		if r.CustomerID == customerID {
			out = append(out, domain.Movement{Kind: domain.MovementCharge, At: r.CreatedAt, Amount: r.Amount})
		}
	}
	for _, p := range f.s.payments {
		if p.CustomerID() == customerID {
			out = append(out, domain.Movement{Kind: domain.MovementPayment, At: p.CreatedAt(), Amount: p.Amount()})
		}
	}
	return out, nil
}

func (f fakeCredit) OpenBalances(_ context.Context, _ uuid.UUID) ([]domain.OpenBalance, error) {
	var out []domain.OpenBalance
	for _, r := range f.s.receivables {
		if r.Status == domain.ReceivableOpen {
			out = append(out, domain.OpenBalance{CustomerID: r.CustomerID, DueDate: r.DueDate, Balance: r.Balance})
		}
	}
	return out, nil
}

// fakeCustomers is the tenant's customers, by name.
type fakeCustomers struct{ names map[uuid.UUID]string }

func (f *fakeCustomers) CheckCustomer(_ context.Context, _, _, customerID uuid.UUID) error {
	if _, ok := f.names[customerID]; !ok {
		return domain.ErrCustomerNotFound
	}
	return nil
}

func (f *fakeCustomers) Names(_ context.Context, _, _ uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	out := map[uuid.UUID]string{}
	for _, id := range ids {
		if n, ok := f.names[id]; ok {
			out[id] = n
		}
	}
	return out, nil
}

// fakeDrawer is the actor's open cash session, or none when closed; without
// one, required says whether the tenant rejects the cash.
type fakeDrawer struct {
	store    *store
	session  uuid.UUID
	closed   bool
	required bool
	reasons  []string
}

func (f *fakeDrawer) ReceiveCash(_ context.Context, _, _ uuid.UUID, amount decimal.Decimal, reason string) (uuid.UUID, error) {
	if f.closed {
		if f.required {
			return uuid.Nil, domain.ErrCashSessionRequired
		}
		return uuid.Nil, nil
	}
	f.store.cashIns = append(f.store.cashIns, amount)
	f.reasons = append(f.reasons, reason)
	return f.session, nil
}
