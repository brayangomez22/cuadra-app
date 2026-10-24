package app_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
)

// In-memory fakes of the identity ports. They copy entities in and out, like
// a database would, so a use case cannot change stored state by mutating an
// entity it did not save.

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time          { return c.now }
func (c *fakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// fakeModules keeps the active modules of each tenant.
type fakeModules struct {
	mu     sync.Mutex
	active map[uuid.UUID][]string
}

func (f *fakeModules) Active(_ context.Context, tenantID uuid.UUID) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.active[tenantID]), nil
}

func (f *fakeModules) activate(tenantID uuid.UUID, ids ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.active[tenantID] = append(f.active[tenantID], ids...)
}

// fakeRegistry is the module registry as identity sees it: the permissions
// each module declares.
type fakeRegistry []app.ModulePermission

func (f fakeRegistry) Permissions() []app.ModulePermission { return f }

// testRegistry declares the permissions of two modules of different kinds of
// business: lots (pharmacies) and work orders (repair shops).
var testRegistry = fakeRegistry{
	{Module: "lots_expiry", Permission: "lots:manage", Roles: []string{"admin", "warehouse"}},
	{Module: "work_orders", Permission: "work_orders:write", Roles: []string{"cashier"}},
}

// testTemplates are the business templates fakeSetup offers.
var testTemplates = []app.BusinessTemplate{
	{ID: "generica", Name: "Negocio general", Description: "Cualquier comercio.", DefaultTaxRate: decimal.RequireFromString("0.19")},
	{ID: "ferreteria", Name: "Ferretería", Description: "Herramientas y construcción.", DefaultTaxRate: decimal.RequireFromString("0.19")},
	{ID: "ropa_calzado", Name: "Ropa y calzado", Description: "Prendas por talla y color.", DefaultTaxRate: decimal.RequireFromString("0.19")},
	{ID: "drogueria", Name: "Droguería", Description: "Medicamentos y cuidado personal.", DefaultTaxRate: decimal.Zero},
}

// fakeSetup offers testTemplates and records the tenants it prepared, with
// their template and whether it ran inside CreateWithOwner's transaction;
// err makes it fail.
type fakeSetup struct {
	mu        sync.Mutex
	tenants   []uuid.UUID
	templates []string
	inTx      []bool
	err       error
}

func (f *fakeSetup) Templates() []app.BusinessTemplate { return slices.Clone(testTemplates) }

func (f *fakeSetup) SetUpTenant(ctx context.Context, tenantID uuid.UUID, template string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	txTenant, ok := ctx.Value(inTx{}).(uuid.UUID)
	f.tenants = append(f.tenants, tenantID)
	f.templates = append(f.templates, template)
	f.inTx = append(f.inTx, ok && txTenant == tenantID)
	return nil
}

// fakeHasher "hashes" by prefixing, and counts verifications to check that
// a login with an unknown email still pays for one.
type fakeHasher struct {
	mu       sync.Mutex
	hashes   int
	verifies int
}

func (h *fakeHasher) Hash(plain string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hashes++
	return "hash:" + plain, nil
}

func (h *fakeHasher) Hashes() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hashes
}

func (h *fakeHasher) Verify(plain, hash string) (bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.verifies++
	if !strings.HasPrefix(hash, "hash:") {
		return false, fmt.Errorf("malformed hash")
	}
	return hash == "hash:"+plain, nil
}

func (h *fakeHasher) Verifies() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.verifies
}

// fakeIssuer issues readable access tokens: "access:<tenant>:<user>:<role>".
type fakeIssuer struct{ clock *fakeClock }

func (i fakeIssuer) Issue(tenantID, userID uuid.UUID, role string) (string, time.Time, error) {
	return fmt.Sprintf("access:%s:%s:%s", tenantID, userID, role), i.clock.Now().Add(15 * time.Minute), nil
}

// fakeLimiter allows every key except the denied ones, and records the keys
// it was asked about.
type fakeLimiter struct {
	mu     sync.Mutex
	denied map[string]bool
	keys   []string
}

func (l *fakeLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.keys = append(l.keys, key)
	if l.denied[key] {
		return false, time.Minute
	}
	return true, 0
}
