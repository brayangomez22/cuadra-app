package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const maxLocationNameLength = 100

// Location is a store or warehouse where stock lives. A tenant has at least
// one; stock, sales and cash registers always reference a location.
type Location struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	name      string
	active    bool
	createdAt time.Time
}

// NewLocation creates an active location.
func NewLocation(tenantID uuid.UUID, name string, now time.Time) (*Location, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return newLocation(LocationSnapshot{ID: id, TenantID: tenantID, Name: name, Active: true, CreatedAt: now})
}

// newLocation checks the invariants shared by NewLocation and RehydrateLocation.
func newLocation(s LocationSnapshot) (*Location, error) {
	if s.ID == uuid.Nil {
		return nil, ErrInvalidLocationID
	}
	if s.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	name, err := normalizeLocationName(s.Name)
	if err != nil {
		return nil, err
	}
	return &Location{id: s.ID, tenantID: s.TenantID, name: name, active: s.Active, createdAt: s.CreatedAt.UTC()}, nil
}

func normalizeLocationName(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > maxLocationNameLength {
		return "", ErrInvalidLocationName
	}
	return s, nil
}

// ID returns the location ID (UUID v7).
func (l *Location) ID() uuid.UUID { return l.id }

// TenantID returns the tenant the location belongs to.
func (l *Location) TenantID() uuid.UUID { return l.tenantID }

// Name returns the location name.
func (l *Location) Name() string { return l.name }

// IsActive reports whether the location is in use.
func (l *Location) IsActive() bool { return l.active }

// CreatedAt returns the creation time in UTC.
func (l *Location) CreatedAt() time.Time { return l.createdAt }

// Rename sets a new name. On error the name is unchanged.
func (l *Location) Rename(name string) error {
	name, err := normalizeLocationName(name)
	if err != nil {
		return err
	}
	l.name = name
	return nil
}

// Deactivate takes the location out of use without deleting it, so its
// movements keep their reference. Idempotent.
func (l *Location) Deactivate() { l.active = false }

// Activate puts an inactive location back in use. Idempotent.
func (l *Location) Activate() { l.active = true }

// LocationSnapshot holds a stored location's fields, for RehydrateLocation.
type LocationSnapshot struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	Active    bool
	CreatedAt time.Time
}

// RehydrateLocation rebuilds a stored location, checking the same invariants
// as NewLocation.
func RehydrateLocation(s LocationSnapshot) (*Location, error) { return newLocation(s) }
