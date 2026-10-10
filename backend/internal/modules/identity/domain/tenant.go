package domain

import (
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

// Tenant is a business using Cuadra (a hardware store). Its fields are
// private so it cannot be put in an invalid state.
type Tenant struct {
	id        uuid.UUID
	name      string
	nit       NIT
	status    TenantStatus
	createdAt time.Time
}

// NewTenant creates an active tenant. The name is trimmed; the NIT is required.
func NewTenant(name string, nit NIT, now time.Time) (*Tenant, error) {
	name, err := normalizeName(name, ErrInvalidTenantName)
	if err != nil {
		return nil, err
	}
	if nit.IsZero() {
		return nil, ErrInvalidNIT
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return &Tenant{id: id, name: name, nit: nit, status: TenantActive, createdAt: now.UTC()}, nil
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

// IsActive reports whether the tenant can operate.
func (t *Tenant) IsActive() bool { return t.status == TenantActive }

// Suspend blocks the tenant (for example, unpaid subscription). Idempotent.
func (t *Tenant) Suspend() { t.status = TenantSuspended }

// Activate reactivates a suspended tenant. Idempotent.
func (t *Tenant) Activate() { t.status = TenantActive }
