package domain

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxUnitCodeLength = 16
	maxUnitNameLength = 50
)

// UnitOfMeasure is a unit a tenant counts and sells its products in ("m",
// "kg", "par"). Each tenant has its own catalog of units. Quantities are
// decimal, so fractional amounts (2.5 m) need no special unit.
type UnitOfMeasure struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	code      string
	name      string
	createdAt time.Time
}

// UnitRef is what a product keeps of its base unit: enough to show it next
// to a quantity without loading the unit.
type UnitRef struct {
	ID   uuid.UUID
	Code string
	Name string
}

// NewUnit creates a unit with its own UUID v7.
func NewUnit(tenantID uuid.UUID, code, name string, now time.Time) (*UnitOfMeasure, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return newUnit(UnitSnapshot{ID: id, TenantID: tenantID, Code: code, Name: name, CreatedAt: now})
}

// newUnit checks the invariants shared by NewUnit and RehydrateUnit.
func newUnit(s UnitSnapshot) (*UnitOfMeasure, error) {
	if s.ID == uuid.Nil {
		return nil, ErrInvalidUnitID
	}
	if s.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	u := &UnitOfMeasure{id: s.ID, tenantID: s.TenantID, createdAt: s.CreatedAt.UTC()}
	if err := u.Update(s.Code, s.Name); err != nil {
		return nil, err
	}
	return u, nil
}

// normalizeUnitCode trims and lowercases code. It is shown next to a
// quantity ("2,5 m"), so it has no spaces and is short.
func normalizeUnitCode(code string) (string, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" || utf8.RuneCountInString(code) > maxUnitCodeLength || strings.ContainsFunc(code, unicode.IsSpace) {
		return "", ErrInvalidUnitCode
	}
	return code, nil
}

// ID returns the unit ID (UUID v7).
func (u *UnitOfMeasure) ID() uuid.UUID { return u.id }

// TenantID returns the tenant the unit belongs to.
func (u *UnitOfMeasure) TenantID() uuid.UUID { return u.tenantID }

// Code returns the code shown next to a quantity ("m").
func (u *UnitOfMeasure) Code() string { return u.code }

// Name returns the unit's name ("Metro").
func (u *UnitOfMeasure) Name() string { return u.name }

// CreatedAt returns the creation time in UTC.
func (u *UnitOfMeasure) CreatedAt() time.Time { return u.createdAt }

// Ref returns what a product keeps of the unit.
func (u *UnitOfMeasure) Ref() UnitRef { return UnitRef{ID: u.id, Code: u.code, Name: u.name} }

// Update replaces the code and the name; on error the unit is unchanged.
func (u *UnitOfMeasure) Update(code, name string) error {
	code, err := normalizeUnitCode(code)
	if err != nil {
		return err
	}
	name, err = normalizeText(name, maxUnitNameLength, ErrInvalidUnitName)
	if err != nil {
		return err
	}
	u.code, u.name = code, name
	return nil
}

// UnitSnapshot holds a stored unit's fields, for RehydrateUnit.
type UnitSnapshot struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Code      string
	Name      string
	CreatedAt time.Time
}

// RehydrateUnit rebuilds a stored unit, checking the same invariants as
// NewUnit so corrupt data surfaces as an error instead of an invalid entity.
func RehydrateUnit(s UnitSnapshot) (*UnitOfMeasure, error) { return newUnit(s) }
