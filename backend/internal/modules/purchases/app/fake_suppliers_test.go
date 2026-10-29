package app_test

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

// fakeSuppliers is an in-memory SupplierRepository.
type fakeSuppliers struct {
	rows map[uuid.UUID]domain.SupplierSnapshot
}

func newFakeSuppliers() *fakeSuppliers {
	return &fakeSuppliers{rows: map[uuid.UUID]domain.SupplierSnapshot{}}
}

func (f *fakeSuppliers) Create(_ context.Context, s *domain.Supplier) error {
	if f.nitTaken(s) {
		return domain.ErrNITTaken
	}
	f.rows[s.ID()] = s.Snapshot()
	return nil
}

func (f *fakeSuppliers) Update(_ context.Context, s *domain.Supplier) error {
	if old, ok := f.rows[s.ID()]; !ok || old.TenantID != s.TenantID() {
		return domain.ErrSupplierNotFound
	}
	if f.nitTaken(s) {
		return domain.ErrNITTaken
	}
	f.rows[s.ID()] = s.Snapshot()
	return nil
}

func (f *fakeSuppliers) nitTaken(s *domain.Supplier) bool {
	for _, r := range f.rows {
		if r.TenantID == s.TenantID() && r.ID != s.ID() && r.NIT == s.NIT().String() {
			return true
		}
	}
	return false
}

func (f *fakeSuppliers) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.Supplier, error) {
	r, ok := f.rows[id]
	if !ok || r.TenantID != tenantID {
		return nil, domain.ErrSupplierNotFound
	}
	return domain.RehydrateSupplier(r)
}

func (f *fakeSuppliers) Search(_ context.Context, flt domain.SupplierFilter) ([]*domain.Supplier, int, error) {
	var out []*domain.Supplier
	for _, r := range f.rows {
		matches := strings.Contains(strings.ToLower(r.Name), strings.ToLower(flt.Text)) ||
			strings.HasPrefix(r.NIT, flt.Text)
		if r.TenantID != flt.TenantID || !matches || (flt.Status == domain.SuppliersActive && !r.Active) ||
			(flt.Status == domain.SuppliersInactive && r.Active) {
			continue
		}
		s, err := domain.RehydrateSupplier(r)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	slices.SortFunc(out, func(a, b *domain.Supplier) int { return strings.Compare(a.Name(), b.Name()) })
	return out, len(out), nil
}
