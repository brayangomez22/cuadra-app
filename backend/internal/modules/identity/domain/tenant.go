package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
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
	id        uuid.UUID
	name      string
	nit       NIT
	status    TenantStatus
	template  string
	createdAt time.Time
}

// businessTemplateFormat is the format of a business template ID (the name
// of its file); which templates exist is up to the use case.
var businessTemplateFormat = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// NewTenant creates an active tenant. The name is trimmed; the NIT and the
// business template it signs up with are required.
func NewTenant(name string, nit NIT, businessTemplate string, now time.Time) (*Tenant, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return newTenant(TenantSnapshot{
		ID: id, Name: name, NIT: nit, Status: TenantActive, BusinessTemplate: businessTemplate, CreatedAt: now,
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
	return &Tenant{
		id: s.ID, name: name, nit: s.NIT, status: s.Status, template: s.BusinessTemplate, createdAt: s.CreatedAt.UTC(),
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

// IsActive reports whether the tenant can operate.
func (t *Tenant) IsActive() bool { return t.status == TenantActive }

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
	CreatedAt        time.Time
}

// RehydrateTenant rebuilds a stored tenant, checking the same invariants as
// NewTenant so corrupt data surfaces as an error instead of an invalid entity.
func RehydrateTenant(s TenantSnapshot) (*Tenant, error) { return newTenant(s) }
