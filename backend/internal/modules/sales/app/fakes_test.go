package app_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

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

// fakeCash keeps cash sessions in memory. Its activity is computed from the
// stored sales as the SQL does: only confirmed sales of the session count.
type fakeCash struct{ *store }

func cloneSession(s *domain.CashSession) *domain.CashSession {
	c, err := domain.RehydrateCashSession(s.Snapshot())
	if err != nil {
		panic(err)
	}
	return c
}

func (f fakeCash) Open(_ context.Context, s *domain.CashSession) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, other := range f.sessions {
		if other.TenantID() == s.TenantID() && other.UserID() == s.UserID() && other.Status() == domain.CashSessionOpen {
			return domain.ErrCashSessionAlreadyOpen
		}
	}
	f.sessions[s.ID()] = cloneSession(s)
	return nil
}

func (f fakeCash) get(tenantID, id uuid.UUID) (*domain.CashSession, error) {
	s, ok := f.sessions[id]
	if !ok || s.TenantID() != tenantID {
		return nil, domain.ErrCashSessionNotFound
	}
	return cloneSession(s), nil
}

func (f fakeCash) Get(_ context.Context, tenantID, id uuid.UUID) (*domain.CashSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.get(tenantID, id)
}

func (f fakeCash) Current(_ context.Context, tenantID, userID uuid.UUID) (*domain.CashSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.sessions {
		if s.TenantID() == tenantID && s.UserID() == userID && s.Status() == domain.CashSessionOpen {
			return cloneSession(s), nil
		}
	}
	return nil, domain.ErrCashSessionNotFound
}

func (f fakeCash) Movements(_ context.Context, tenantID, id uuid.UUID) ([]domain.CashMovement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.CashMovement
	for _, m := range f.movements {
		if m.TenantID() == tenantID && m.SessionID() == id {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f fakeCash) AddMovement(_ context.Context, tenantID, id uuid.UUID, record func(*domain.CashSession) (domain.CashMovement, error)) (domain.CashMovement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, err := f.get(tenantID, id)
	if err != nil {
		return domain.CashMovement{}, err
	}
	m, err := record(s)
	if err != nil {
		return domain.CashMovement{}, err
	}
	f.movements = append(f.movements, m)
	return m, nil
}

func (f fakeCash) EnsureOpen(_ context.Context, tenantID, id uuid.UUID) error {
	if f.onEnsureOpen != nil {
		f.onEnsureOpen()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensured = append(f.ensured, id)
	s, err := f.get(tenantID, id)
	if err != nil {
		return err
	}
	if s.Status() != domain.CashSessionOpen {
		return domain.ErrCashSessionClosed
	}
	return nil
}

func (f fakeCash) Close(_ context.Context, tenantID, id uuid.UUID, closeFn func(*domain.CashSession, domain.CashActivity) error) (*domain.CashSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, err := f.get(tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := closeFn(s, f.activity(id)); err != nil {
		return nil, err
	}
	f.sessions[id] = cloneSession(s)
	return s, nil
}

// activity adds up the confirmed sales and the movements of a session.
func (f fakeCash) activity(id uuid.UUID) domain.CashActivity {
	a := domain.CashActivity{CashReceived: decimal.Zero, ChangeGiven: decimal.Zero, CashIn: decimal.Zero, CashOut: decimal.Zero}
	byMethod := map[domain.PaymentMethod]decimal.Decimal{}
	for _, sale := range f.sales {
		if sid := sale.CashSessionID(); sid == nil || *sid != id || sale.Status() != domain.SaleConfirmed {
			continue
		}
		a.Sales++
		a.ChangeGiven = a.ChangeGiven.Add(sale.Change())
		byMethod[domain.PaymentCash] = byMethod[domain.PaymentCash].Sub(sale.Change())
		for _, p := range sale.Payments() {
			if p.Method() == domain.PaymentCash {
				a.CashReceived = a.CashReceived.Add(p.Amount())
			}
			byMethod[p.Method()] = byMethod[p.Method()].Add(p.Amount())
		}
	}
	for _, m := range []domain.PaymentMethod{domain.PaymentCash, domain.PaymentCard, domain.PaymentNequi, domain.PaymentDaviplata, domain.PaymentTransfer} {
		if amount, ok := byMethod[m]; ok {
			a.ByMethod = append(a.ByMethod, domain.MethodTotal{Method: m, Amount: amount})
		}
	}
	for _, m := range f.movements {
		if m.SessionID() != id {
			continue
		}
		if m.Kind() == domain.CashIn {
			a.CashIn = a.CashIn.Add(m.Amount())
		} else {
			a.CashOut = a.CashOut.Add(m.Amount())
		}
	}
	return a
}

type fakePolicies struct{ *store }

func (f fakePolicies) Get(_ context.Context, tenantID uuid.UUID) (domain.SalesPolicy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.policies[tenantID], nil
}

var testNow = time.Date(2026, 10, 10, 9, 30, 0, 0, time.FixedZone("COT", -5*3600))

type env struct {
	svc     *app.Service
	store   *store
	actor   app.Actor
	spans   *tracetest.SpanRecorder
	metrics *sdkmetric.ManualReader
}

// newEnv builds a Service on the fakes; rules go before the base price rule.
func newEnv(t *testing.T, rules ...app.PriceRule) *env {
	t.Helper()
	e := &env{
		store: &store{
			variants: map[catalogKey]app.VariantInfo{},
			policies: map[uuid.UUID]domain.SalesPolicy{},
			sales:    map[uuid.UUID]*domain.Sale{},
			numbers:  map[uuid.UUID]int64{},
			sessions: map[uuid.UUID]*domain.CashSession{},
		},
		actor:   app.Actor{TenantID: newID(), UserID: newID()},
		spans:   tracetest.NewSpanRecorder(),
		metrics: sdkmetric.NewManualReader(),
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	chain, err := app.NewPriceChain(rules...)
	require.NoError(t, err)
	svc, err := app.NewService(app.Deps{
		Catalog:        fakeCatalog{e.store},
		Prices:         recordingPrices{e.store, chain},
		Stock:          fakeStock{e.store},
		Sales:          fakeSales{e.store},
		Policies:       fakePolicies{e.store},
		CashSessions:   fakeCash{e.store},
		Now:            func() time.Time { return testNow },
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	// The sales tests predate the cash sessions: their tenant sells without
	// one. The cash tests require it with requireCashSession.
	e.store.policies[e.actor.TenantID] = domain.SalesPolicy{CashSessionOptional: true}
	return e
}

// requireCashSession makes the tenant require an open cash session to sell,
// the default of a tenant without settings.
func (e *env) requireCashSession() {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	p := e.store.policies[e.actor.TenantID]
	p.CashSessionOptional = false
	e.store.policies[e.actor.TenantID] = p
}

// addProduct puts a simple product in the actor's catalog and returns its
// default variant, whose id is not the product's.
func (e *env) addProduct(name, price, rate string, active bool) uuid.UUID {
	productID, variantID := newID(), newID()
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	e.store.variants[catalogKey{e.actor.TenantID, variantID}] = app.VariantInfo{
		ID: variantID, ProductID: productID, ProductName: name, Price: dec(price), TaxRate: dec(rate), Active: active,
	}
	return variantID
}

// addVariant adds a variant with label to the product of variant sibling
// and returns it.
func (e *env) addVariant(sibling uuid.UUID, label, price string, active bool) uuid.UUID {
	id := newID()
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	v := e.store.variants[catalogKey{e.actor.TenantID, sibling}]
	v.ID, v.Label, v.Price, v.Active = id, label, dec(price), active
	e.store.variants[catalogKey{e.actor.TenantID, id}] = v
	return id
}

// productOf returns the product of a variant of the actor's catalog.
func (e *env) productOf(variantID uuid.UUID) uuid.UUID {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	return e.store.variants[catalogKey{e.actor.TenantID, variantID}].ProductID
}

// setMaxDiscount sets the tenant's discount limit for users without
// sales:discount.
func (e *env) setMaxDiscount(t *testing.T, fraction string) {
	t.Helper()
	limit, err := domain.NewDiscountLimit(dec(fraction))
	require.NoError(t, err)
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	p := e.store.policies[e.actor.TenantID]
	p.MaxDiscount = limit
	e.store.policies[e.actor.TenantID] = p
}

func (e *env) storedSales() int {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	return len(e.store.sales)
}

// span returns the attributes of the only ended span named name.
func (e *env) span(t *testing.T, name string) (sdktrace.ReadOnlySpan, map[attribute.Key]attribute.Value) {
	t.Helper()
	var found []sdktrace.ReadOnlySpan
	for _, s := range e.spans.Ended() {
		if s.Name() == name {
			found = append(found, s)
		}
	}
	require.Len(t, found, 1, "spans named %s", name)
	attrs := map[attribute.Key]attribute.Value{}
	for _, kv := range found[0].Attributes() {
		attrs[kv.Key] = kv.Value
	}
	return found[0], attrs
}

func (e *env) collect(t *testing.T) []metricdata.Metrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, e.metrics.Collect(t.Context(), &rm))
	var out []metricdata.Metrics
	for _, sm := range rm.ScopeMetrics {
		out = append(out, sm.Metrics...)
	}
	return out
}

// counter returns the sum of the data points of an int64 counter.
func (e *env) counter(t *testing.T, name string) int64 {
	t.Helper()
	var total int64
	for _, m := range e.collect(t) {
		if m.Name != name {
			continue
		}
		sum, ok := m.Data.(metricdata.Sum[int64])
		require.True(t, ok, "%s is not an int64 counter", name)
		for _, dp := range sum.DataPoints {
			total += dp.Value
		}
	}
	return total
}

// histogram returns the count and sum of a float64 histogram.
func (e *env) histogram(t *testing.T, name string) (count uint64, sum float64) {
	t.Helper()
	for _, m := range e.collect(t) {
		if m.Name != name {
			continue
		}
		h, ok := m.Data.(metricdata.Histogram[float64])
		require.True(t, ok, "%s is not a float64 histogram", name)
		for _, dp := range h.DataPoints {
			count += dp.Count
			sum += dp.Sum
		}
	}
	return count, sum
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }
