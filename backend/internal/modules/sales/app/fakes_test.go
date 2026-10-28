package app_test

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// In-memory fakes of the sales ports.

// catalogKey is an id (of a variant or of a product) in a tenant.
type catalogKey struct{ tenantID, id uuid.UUID }

type reservation struct {
	tenantID, userID, locationID, saleID uuid.UUID
	lines                                []app.StockLine
}

type store struct {
	mu       sync.Mutex
	variants map[catalogKey]app.VariantInfo
	policies map[uuid.UUID]domain.SalesPolicy
	sales    map[uuid.UUID]*domain.Sale
	reserved []reservation
	// reserveErr, when set, makes the next reservations fail.
	reserveErr error
	// reversed records the sales whose stock was given back; reverseErr,
	// when set, makes the next reversals fail.
	reversed   []uuid.UUID
	reverseErr error
	// numbers is the last number given at each location.
	numbers map[uuid.UUID]int64
	// filters records the filters List received.
	filters []domain.SaleFilter
	// catalogCalls records the variant ids each catalog lookup received.
	catalogCalls [][]uuid.UUID
	// events records, in order, when prices were resolved ("resolve") and
	// when the sale's transaction began ("save").
	events []string
	// sessions and movements are the cash sessions and their movements.
	sessions  map[uuid.UUID]*domain.CashSession
	movements []domain.CashMovement
	// ensured records the sessions EnsureOpen checked; onEnsureOpen, when
	// set, runs before the check (to close the session meanwhile).
	ensured      []uuid.UUID
	onEnsureOpen func()
	// customers are the customers of each tenant and whether they are
	// active.
	customers map[catalogKey]bool
}

type fakeCustomers struct{ *store }

func (f fakeCustomers) CheckCustomer(_ context.Context, tenantID, _, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	active, ok := f.customers[catalogKey{tenantID, id}]
	switch {
	case !ok:
		return domain.ErrCustomerNotFound
	case !active:
		return domain.ErrCustomerInactive
	}
	return nil
}

type fakeCatalog struct{ *store }

func (f fakeCatalog) Variants(_ context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]app.VariantInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.catalogCalls = append(f.catalogCalls, ids)
	var out []app.VariantInfo
	for _, id := range ids {
		if v, ok := f.variants[catalogKey{tenantID, id}]; ok {
			out = append(out, v)
		}
	}
	return out, nil
}

// recordingPrices is the base price chain, recording when it resolves.
type recordingPrices struct {
	*store
	chain *app.PriceChain
}

func (r recordingPrices) Resolve(ctx context.Context, q app.PriceQuery) (app.ResolvedPrice, error) {
	r.mu.Lock()
	r.events = append(r.events, "resolve")
	r.mu.Unlock()
	return r.chain.Resolve(ctx, q)
}

type fakeStock struct{ *store }

func (f fakeStock) ReserveSale(_ context.Context, tenantID, userID, locationID, saleID uuid.UUID, lines []app.StockLine) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.reserveErr != nil {
		return f.reserveErr
	}
	f.reserved = append(f.reserved, reservation{tenantID, userID, locationID, saleID, lines})
	return nil
}

func (f fakeStock) ReverseSale(_ context.Context, _, _, saleID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.reverseErr != nil {
		return f.reverseErr
	}
	f.reversed = append(f.reversed, saleID)
	return nil
}

// clone copies a sale in or out of the fake, as a database would.
func clone(s *domain.Sale) *domain.Sale {
	c, err := domain.RehydrateSale(s.Snapshot())
	if err != nil {
		panic(err)
	}
	return c
}

// fakeSales stores the sale only if reserve succeeds, as one transaction
// would, and numbers it per location.
type fakeSales struct{ *store }

func (f fakeSales) SaveConfirmed(ctx context.Context, s *domain.Sale, reserve func(context.Context) error) error {
	f.mu.Lock()
	f.events = append(f.events, "save")
	f.mu.Unlock()
	if err := reserve(ctx); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := s.AssignNumber(f.numbers[s.LocationID()] + 1); err != nil {
		return err
	}
	f.numbers[s.LocationID()]++
	f.sales[s.ID()] = clone(s)
	return nil
}

func (f fakeSales) Get(_ context.Context, tenantID, id uuid.UUID) (*domain.Sale, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sales[id]
	if !ok || s.TenantID() != tenantID {
		return nil, domain.ErrSaleNotFound
	}
	return clone(s), nil
}

func (f fakeSales) List(_ context.Context, tenantID uuid.UUID, filter domain.SaleFilter) (domain.SalePage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.filters = append(f.filters, filter)
	var page domain.SalePage
	for _, s := range f.sales {
		at := *s.ConfirmedAt()
		if s.TenantID() != tenantID || at.Before(filter.From) || !at.Before(filter.To) ||
			filter.LocationID != nil && s.LocationID() != *filter.LocationID {
			continue
		}
		page.Items = append(page.Items, domain.SummaryOf(s))
	}
	page.Total = len(page.Items)
	return page, nil
}

// Void changes a copy and stores it only if change succeeds, as one
// transaction would.
func (f fakeSales) Void(ctx context.Context, tenantID, id uuid.UUID, change func(context.Context, *domain.Sale) error) (*domain.Sale, error) {
	s, err := f.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := change(ctx, s); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sales[id] = clone(s)
	return s, nil
}

type fakePolicies struct{ *store }

func (f fakePolicies) Get(_ context.Context, tenantID uuid.UUID) (domain.SalesPolicy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.policies[tenantID], nil
}
