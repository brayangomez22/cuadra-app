package app_test

import (
	"context"
	"maps"
	"slices"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// countStore holds the fake counts, lines and entries.
type countStore struct {
	counts  map[uuid.UUID]domain.StockCountSnapshot
	lines   map[countLineKey]domain.CountLine
	entries []countedEntry
	// categories are the variants of each category the fake catalog knows.
	categories map[uuid.UUID][]uuid.UUID
}

type countLineKey struct{ countID, variantID uuid.UUID }

type countedEntry struct {
	domain.CountEntrySnapshot
	round int
}

func newCountStore() countStore {
	return countStore{
		counts:     map[uuid.UUID]domain.StockCountSnapshot{},
		lines:      map[countLineKey]domain.CountLine{},
		categories: map[uuid.UUID][]uuid.UUID{},
	}
}

func countSnapshot(c *domain.StockCount) domain.StockCountSnapshot {
	s := domain.StockCountSnapshot{
		StockCountSpec: domain.StockCountSpec{
			ID: c.ID(), TenantID: c.TenantID(), LocationID: c.LocationID(), CategoryIDs: c.CategoryIDs(),
			Mode: c.Mode(), StartedBy: c.StartedBy(), StartedAt: c.StartedAt(),
		},
		Status: c.Status(), ClosedBy: c.ClosedBy(), ClosedAt: c.ClosedAt(),
	}
	if summary, ok := c.Summary(); ok {
		s.Summary = &summary
	}
	return s
}

// fakeCounts is the StockCountRepository.
type fakeCounts struct{ *store }

func (f fakeCounts) Start(_ context.Context, c *domain.StockCount, variantIDs []uuid.UUID) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if l, ok := f.locations[c.LocationID()]; !ok || l.TenantID() != c.TenantID() {
		return 0, domain.ErrLocationNotFound
	}
	for _, other := range f.counts {
		if other.TenantID == c.TenantID() && other.LocationID == c.LocationID() && other.Status == domain.CountOpen {
			return 0, domain.ErrCountInProgress
		}
	}
	if !c.IsPartial() {
		for key := range f.levels {
			if key.tenantID == c.TenantID() && key.locationID == c.LocationID() {
				variantIDs = append(variantIDs, key.variantID)
			}
		}
	}
	for _, v := range variantIDs {
		level := f.levels[levelKey{c.TenantID(), v, c.LocationID()}]
		f.lines[countLineKey{c.ID(), v}] = domain.CountLine{
			VariantID: v, StartQuantity: f.kardexSum(c, v, nil), StartCost: level.AverageCost, Round: 1,
		}
	}
	f.counts[c.ID()] = countSnapshot(c)
	return len(variantIDs), nil
}

// kardexSum adds the movements of the variant at the count's location up to
// its start, or after its start and up to until.
func (s *store) kardexSum(c *domain.StockCount, variantID uuid.UUID, until *countedEntry) decimal.Decimal {
	total := decimal.Zero
	for _, m := range s.movements {
		if m.TenantID() != c.TenantID() || m.VariantID() != variantID || m.LocationID() != c.LocationID() {
			continue
		}
		before := !m.OccurredAt().After(c.StartedAt())
		during := until != nil && !before && !m.OccurredAt().After(until.CountedAt)
		if (until == nil && before) || during {
			total = total.Add(m.Quantity())
		}
	}
	return total
}

func (f fakeCounts) Get(_ context.Context, tenantID, id uuid.UUID) (*domain.StockCount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.get(tenantID, id)
}

func (s *store) get(tenantID, id uuid.UUID) (*domain.StockCount, error) {
	snap, ok := s.counts[id]
	if !ok || snap.TenantID != tenantID {
		return nil, domain.ErrCountNotFound
	}
	return domain.RehydrateStockCount(snap)
}

func (f fakeCounts) List(_ context.Context, tenantID uuid.UUID, filter domain.CountFilter) (domain.StockCountPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var page domain.StockCountPage
	for _, id := range slices.SortedFunc(maps.Keys(f.counts), compareUUID) {
		snap := f.counts[id]
		if snap.TenantID != tenantID || (filter.LocationID != nil && *filter.LocationID != snap.LocationID) ||
			(filter.Status != nil && *filter.Status != snap.Status) {
			continue
		}
		c, err := domain.RehydrateStockCount(snap)
		if err != nil {
			return page, err
		}
		page.Items = append(page.Items, c)
	}
	page.Total = len(page.Items)
	return page, nil
}

func (f fakeCounts) Progress(_ context.Context, tenantID, id uuid.UUID) (domain.CountProgress, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, err := f.get(tenantID, id)
	if err != nil {
		return domain.CountProgress{}, err
	}
	var p domain.CountProgress
	for _, l := range f.sheet(c) {
		p.Lines++
		if len(l.Counts) > 0 {
			p.Counted++
		}
	}
	return p, nil
}

func (f fakeCounts) Sheet(_ context.Context, c *domain.StockCount) ([]domain.SheetLine, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sheet(c), nil
}

// sheet builds the lines of c by variant, like the repository's query.
func (s *store) sheet(c *domain.StockCount) []domain.SheetLine {
	var out []domain.SheetLine
	for key, line := range s.lines {
		if key.countID != c.ID() {
			continue
		}
		sl := domain.SheetLine{CountLine: line, Movements: decimal.Zero}
		totals := map[uuid.UUID]decimal.Decimal{}
		var last *countedEntry
		for i, e := range s.entries {
			if e.CountID == c.ID() && e.VariantID == line.VariantID && e.round == line.Round {
				totals[e.CountedBy] = totals[e.CountedBy].Add(e.Quantity)
				if last == nil || e.CountedAt.After(last.CountedAt) {
					last = &s.entries[i]
				}
			}
		}
		for _, user := range slices.SortedFunc(maps.Keys(totals), compareUUID) {
			sl.Counts = append(sl.Counts, domain.CounterTotal{UserID: user, Quantity: totals[user]})
		}
		if last != nil {
			sl.Movements = s.kardexSum(c, line.VariantID, last)
		}
		out = append(out, sl)
	}
	slices.SortFunc(out, func(a, b domain.SheetLine) int { return compareUUID(a.VariantID, b.VariantID) })
	return out
}

func compareUUID(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) }

func (f fakeCounts) Recount(_ context.Context, _, id uuid.UUID, variantIDs []uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, v := range variantIDs {
		if _, ok := f.lines[countLineKey{id, v}]; !ok {
			return domain.ErrVariantNotInCount
		}
	}
	for _, v := range variantIDs {
		line := f.lines[countLineKey{id, v}]
		line.Round++
		f.lines[countLineKey{id, v}] = line
	}
	return nil
}

// Close runs fn and, if it fails, puts the stock back as it was, like a
// rolled back transaction.
func (f fakeCounts) Close(ctx context.Context, tenantID, id uuid.UUID, fn func(context.Context, *domain.StockCount) error) error {
	f.mu.Lock()
	c, err := f.get(tenantID, id)
	levels, movements := maps.Clone(f.levels), slices.Clone(f.movements)
	f.mu.Unlock()
	if err != nil {
		return err
	}
	if err := fn(ctx, c); err != nil {
		f.mu.Lock()
		f.levels, f.movements = levels, movements
		f.mu.Unlock()
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.counts[id] = countSnapshot(c)
	return nil
}
