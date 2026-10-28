package app_test

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
)

// store keeps the fakes' data, like the database would: per tenant, with
// unique documents and segment names, and segments referenced by customers.
type store struct {
	mu        sync.Mutex
	customers map[uuid.UUID]domain.CustomerSnapshot
	segments  map[uuid.UUID]domain.SegmentSnapshot
}

type fakeCustomers struct{ *store }

func (f fakeCustomers) Create(_ context.Context, c *domain.Customer) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.save(c.Snapshot())
}

func (f fakeCustomers) Update(_ context.Context, c *domain.Customer) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if old, ok := f.customers[c.ID()]; !ok || old.TenantID != c.TenantID() {
		return domain.ErrCustomerNotFound
	}
	return f.save(c.Snapshot())
}

// save checks the unique document and the segment's foreign key.
func (f fakeCustomers) save(s domain.CustomerSnapshot) error {
	for _, other := range f.customers {
		if other.ID != s.ID && other.TenantID == s.TenantID &&
			other.DocumentType == s.DocumentType && other.DocumentNumber == s.DocumentNumber {
			doc, _ := domain.ParseDocument(s.DocumentType, s.DocumentNumber)
			return &domain.DocumentTakenError{Document: doc}
		}
	}
	if s.SegmentID != nil {
		if seg, ok := f.segments[*s.SegmentID]; !ok || seg.TenantID != s.TenantID {
			return domain.ErrSegmentNotFound
		}
	}
	f.customers[s.ID] = s
	return nil
}

func (f fakeCustomers) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.customers[id]
	if !ok || s.TenantID != tenantID {
		return nil, domain.ErrCustomerNotFound
	}
	return domain.RehydrateCustomer(s)
}

// Search matches the text as a substring of the name or the document; the
// fake does not rank or fold accents, which the integration tests cover.
func (f fakeCustomers) Search(_ context.Context, flt domain.CustomerFilter) ([]*domain.Customer, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var all []*domain.Customer
	for _, s := range f.customers {
		if !matches(s, flt) {
			continue
		}
		c, err := domain.RehydrateCustomer(s)
		if err != nil {
			return nil, 0, err
		}
		all = append(all, c)
	}
	slices.SortFunc(all, func(a, b *domain.Customer) int { return strings.Compare(a.Name(), b.Name()) })
	total := len(all)
	start, end := min(flt.Offset, total), min(flt.Offset+flt.Limit, total)
	return all[start:end], total, nil
}

func matches(s domain.CustomerSnapshot, f domain.CustomerFilter) bool {
	text := strings.ToLower(f.Text)
	switch {
	case s.TenantID != f.TenantID:
		return false
	case f.Status == domain.StatusActive && !s.Active, f.Status == domain.StatusInactive && s.Active:
		return false
	case f.SegmentID != nil && (s.SegmentID == nil || *s.SegmentID != *f.SegmentID):
		return false
	}
	return text == "" || strings.Contains(strings.ToLower(s.Name), text) || strings.Contains(s.DocumentNumber, text)
}

type fakeSegments struct{ *store }

func (f fakeSegments) Create(_ context.Context, s *domain.Segment) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.save(s)
}

func (f fakeSegments) Update(_ context.Context, s *domain.Segment) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if old, ok := f.segments[s.ID()]; !ok || old.TenantID != s.TenantID() {
		return domain.ErrSegmentNotFound
	}
	return f.save(s)
}

func (f fakeSegments) save(s *domain.Segment) error {
	for _, other := range f.segments {
		if other.ID != s.ID() && other.TenantID == s.TenantID() && strings.EqualFold(other.Name, s.Name()) {
			return domain.ErrSegmentNameTaken
		}
	}
	f.segments[s.ID()] = domain.SegmentSnapshot{ID: s.ID(), TenantID: s.TenantID(), Name: s.Name(), CreatedAt: s.CreatedAt()}
	return nil
}

func (f fakeSegments) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Segment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.segments[id]
	if !ok || s.TenantID != tenantID {
		return nil, domain.ErrSegmentNotFound
	}
	return domain.RehydrateSegment(s)
}

func (f fakeSegments) List(_ context.Context, tenantID uuid.UUID) ([]*domain.Segment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*domain.Segment
	for _, s := range f.segments {
		if s.TenantID == tenantID {
			seg, err := domain.RehydrateSegment(s)
			if err != nil {
				return nil, err
			}
			out = append(out, seg)
		}
	}
	slices.SortFunc(out, func(a, b *domain.Segment) int { return strings.Compare(a.Name(), b.Name()) })
	return out, nil
}

func (f fakeSegments) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.segments[id]; !ok || s.TenantID != tenantID {
		return domain.ErrSegmentNotFound
	}
	for _, c := range f.customers {
		if c.SegmentID != nil && *c.SegmentID == id {
			return domain.ErrSegmentInUse
		}
	}
	delete(f.segments, id)
	return nil
}
