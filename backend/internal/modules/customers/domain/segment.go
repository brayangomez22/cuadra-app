package domain

import (
	"time"

	"github.com/google/uuid"
)

const maxSegmentNameLength = 60

// Segment is a group of customers the tenant defines ("Público", "Mayorista",
// "Convenio"...). A tenant starts with its business template's segments and
// can change them; the core gives them no meaning of their own.
type Segment struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	name      string
	createdAt time.Time
}

// NewSegment creates a segment with its own UUID v7.
func NewSegment(tenantID uuid.UUID, name string, now time.Time) (*Segment, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return RehydrateSegment(SegmentSnapshot{ID: id, TenantID: tenantID, Name: name, CreatedAt: now})
}

// ID returns the segment ID (UUID v7).
func (s *Segment) ID() uuid.UUID { return s.id }

// TenantID returns the tenant the segment belongs to.
func (s *Segment) TenantID() uuid.UUID { return s.tenantID }

// Name returns the segment's name.
func (s *Segment) Name() string { return s.name }

// CreatedAt returns the creation time in UTC.
func (s *Segment) CreatedAt() time.Time { return s.createdAt }

// Rename changes the name; on error the segment is unchanged. Its customers
// stay in it.
func (s *Segment) Rename(name string) error {
	name, err := normalizeText(name, maxSegmentNameLength, ErrInvalidSegmentName)
	if err != nil {
		return err
	}
	s.name = name
	return nil
}

// SegmentSnapshot holds a stored segment's fields, for RehydrateSegment.
type SegmentSnapshot struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	CreatedAt time.Time
}

// RehydrateSegment rebuilds a stored segment, checking the same invariants
// as NewSegment.
func RehydrateSegment(snap SegmentSnapshot) (*Segment, error) {
	if snap.ID == uuid.Nil {
		return nil, ErrInvalidSegmentID
	}
	if snap.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	s := &Segment{id: snap.ID, tenantID: snap.TenantID, createdAt: snap.CreatedAt.UTC()}
	if err := s.Rename(snap.Name); err != nil {
		return nil, err
	}
	return s, nil
}
