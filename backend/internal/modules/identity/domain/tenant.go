package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const maxNameLength = 200

// TenantStatus is the lifecycle state of a tenant.
type TenantStatus string

// Tenant statuses: only active tenants can operate.
const (
	TenantActive    TenantStatus = "active"
	TenantSuspended TenantStatus = "suspended"
)

// Tenant is a business using Cuadra (a shop, a pharmacy, a hardware
// store...). Its fields are private so it cannot be put in an invalid state.
type Tenant struct {
	id       uuid.UUID
	name     string
	nit      NIT
	status   TenantStatus
	template string
	// defaultTaxRate is the IVA rate a new product starts with.
	defaultTaxRate decimal.Decimal
	// demo marks the public demonstration tenants, which are reset every
	// night and never reach real third parties (DIAN, messages).
	demo      bool
	createdAt time.Time
}

// businessTemplateFormat is the format of a business template ID (the name
// of its file); which templates exist is up to the use case.
var businessTemplateFormat = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// NewTenant creates an active tenant. The name is trimmed; the NIT and the
// business template it signs up with are required. defaultTaxRate is the
// IVA rate its template proposes to new products: a fraction from 0 to
// below 1 with up to 4 decimals, as a product's rate.
func NewTenant(name string, nit NIT, businessTemplate string, defaultTaxRate decimal.Decimal, now time.Time) (*Tenant, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return newTenant(TenantSnapshot{
		ID: id, Name: name, NIT: nit, Status: TenantActive, BusinessTemplate: businessTemplate,
		DefaultTaxRate: defaultTaxRate, CreatedAt: now,
	})
}

// newTenant checks the invariants shared by NewTenant and RehydrateTenant.
func newTenant(s TenantSnapshot) (*Tenant, error) {
	if s.ID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	name, err := normalizeName(s.Name, ErrInvalidTenantName)
	if err != nil {
		return nil, err
	}
	if s.NIT.IsZero() {
		return nil, ErrInvalidNIT
	}
	if s.Status != TenantActive && s.Status != TenantSuspended {
		return nil, ErrInvalidTenantStatus
	}
	if !businessTemplateFormat.MatchString(s.BusinessTemplate) {
		return nil, ErrUnknownBusinessTemplate
	}
	if err := checkDefaultTaxRate(s.DefaultTaxRate); err != nil {
		return nil, err
	}
	return &Tenant{
		id: s.ID, name: name, nit: s.NIT, status: s.Status, template: s.BusinessTemplate,
		defaultTaxRate: s.DefaultTaxRate, demo: s.Demo, createdAt: s.CreatedAt.UTC(),
	}, nil
}

// normalizeName trims s and checks it is not empty nor longer than maxNameLength.
func normalizeName(s string, errInvalid error) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > maxNameLength {
		return "", errInvalid
	}
	return s, nil
}

// ID returns the tenant ID (UUID v7).
func (t *Tenant) ID() uuid.UUID { return t.id }

// Name returns the business name.
func (t *Tenant) Name() string { return t.name }

// NIT returns the tax ID.
func (t *Tenant) NIT() NIT { return t.nit }

// Status returns the lifecycle state.
func (t *Tenant) Status() TenantStatus { return t.status }

// CreatedAt returns the creation time in UTC.
func (t *Tenant) CreatedAt() time.Time { return t.createdAt }

// BusinessTemplate returns the ID of the template the tenant signed up with.
func (t *Tenant) BusinessTemplate() string { return t.template }

// DefaultTaxRate returns the IVA rate proposed to a new product.
func (t *Tenant) DefaultTaxRate() decimal.Decimal { return t.defaultTaxRate }

// IsActive reports whether the tenant can operate.
func (t *Tenant) IsActive() bool { return t.status == TenantActive }

// IsDemo reports whether the tenant is a demonstration tenant.
func (t *Tenant) IsDemo() bool { return t.demo }

// MarkDemo makes the tenant a demonstration tenant, which can then be
// purged. Only the demo generator does it, before storing the tenant.
func (t *Tenant) MarkDemo() { t.demo = true }

// Suspend blocks the tenant (for example, unpaid subscription). Idempotent.
func (t *Tenant) Suspend() { t.status = TenantSuspended }

// Activate reactivates a suspended tenant. Idempotent.
func (t *Tenant) Activate() { t.status = TenantActive }

// TenantSnapshot holds a stored tenant's fields, for RehydrateTenant.
type TenantSnapshot struct {
	ID               uuid.UUID
	Name             string
	NIT              NIT
	Status           TenantStatus
	BusinessTemplate string
	DefaultTaxRate   decimal.Decimal
	Demo             bool
	CreatedAt        time.Time
}

// RehydrateTenant rebuilds a stored tenant, checking the same invariants as
// NewTenant so corrupt data surfaces as an error instead of an invalid entity.
func RehydrateTenant(s TenantSnapshot) (*Tenant, error) { return newTenant(s) }

// SetDefaultTaxRate changes the IVA rate proposed to new products; the
// products that exist keep theirs. An invalid rate changes nothing.
func (t *Tenant) SetDefaultTaxRate(rate decimal.Decimal) error {
	if err := checkDefaultTaxRate(rate); err != nil {
		return err
	}
	t.defaultTaxRate = rate
	return nil
}

// checkDefaultTaxRate checks rate is a fraction from 0 to below 1 with up to
// 4 decimals, as a product's rate.
func checkDefaultTaxRate(rate decimal.Decimal) error {
	if rate.IsNegative() || rate.GreaterThanOrEqual(decimal.NewFromInt(1)) || !rate.Equal(rate.Truncate(4)) {
		return ErrInvalidDefaultTaxRate
	}
	return nil
}
