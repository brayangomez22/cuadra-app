package app_test

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func entrySnapshot(e *domain.CountEntry) domain.CountEntrySnapshot {
	return domain.CountEntrySnapshot{
		ID: e.ID(), TenantID: e.TenantID(), CountID: e.CountID(), VariantID: e.VariantID(),
		Quantity: e.Quantity(), CountedBy: e.CountedBy(), CountedAt: e.CountedAt(),
	}
}

// fakeEntries is the CountEntryRepository.
type fakeEntries struct{ *store }

func (f fakeEntries) Add(_ context.Context, e *domain.CountEntry, addLine bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, err := f.get(e.TenantID(), e.CountID())
	if err != nil {
		return err
	}
	if err := c.CheckOpen(); err != nil {
		return err
	}
	if v, ok := f.variants[e.VariantID()]; !ok || v.tenantID != e.TenantID() {
		return domain.ErrVariantNotFound
	}
	key := countLineKey{e.CountID(), e.VariantID()}
	line, ok := f.lines[key]
	if !ok && !addLine {
		return domain.ErrVariantNotInCount
	}
	if !ok {
		line = domain.CountLine{VariantID: e.VariantID(), StartQuantity: decimal.Zero, StartCost: decimal.Zero, Round: 1}
		f.lines[key] = line
	}
	f.entries = append(f.entries, countedEntry{CountEntrySnapshot: entrySnapshot(e), round: line.Round})
	return nil
}

func (f fakeEntries) Get(_ context.Context, tenantID, countID, id uuid.UUID) (*domain.CountEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, e := range f.entries {
		if e.ID == id && e.CountID == countID && e.TenantID == tenantID {
			return domain.RehydrateCountEntry(e.CountEntrySnapshot)
		}
	}
	return nil, domain.ErrCountEntryNotFound
}

func (f fakeEntries) Delete(_ context.Context, e *domain.CountEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := slices.IndexFunc(f.entries, func(s countedEntry) bool { return s.ID == e.ID() })
	if i < 0 {
		return domain.ErrCountEntryNotFound
	}
	f.entries = slices.Delete(f.entries, i, i+1)
	return nil
}

func (f fakeEntries) List(_ context.Context, tenantID uuid.UUID, filter domain.EntryFilter) (domain.CountEntryPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var page domain.CountEntryPage
	for _, e := range slices.Backward(f.entries) {
		line := f.lines[countLineKey{e.CountID, e.VariantID}]
		if e.TenantID != tenantID || e.CountID != filter.CountID || e.round != line.Round ||
			(filter.VariantID != nil && *filter.VariantID != e.VariantID) ||
			(filter.CountedBy != nil && *filter.CountedBy != e.CountedBy) {
			continue
		}
		entry, err := domain.RehydrateCountEntry(e.CountEntrySnapshot)
		if err != nil {
			return page, err
		}
		page.Items = append(page.Items, entry)
	}
	page.Total = len(page.Items)
	return page, nil
}

// VariantsInCategories returns the variants of the actor's tenant in the
// fake categories; ErrCategoryNotFound for a category it does not know.
func (f fakeVariants) VariantsInCategories(_ context.Context, tenantID uuid.UUID, categoryIDs []uuid.UUID) ([]uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []uuid.UUID
	for _, c := range categoryIDs {
		variants, ok := f.categories[c]
		if !ok {
			return nil, domain.ErrCategoryNotFound
		}
		for _, v := range variants {
			if f.variants[v].tenantID == tenantID {
				out = append(out, v)
			}
		}
	}
	return out, nil
}
