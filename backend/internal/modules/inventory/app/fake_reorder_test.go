package app_test

import (
	"context"
	"slices"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// fakeReorder keeps the reorder rules next to the fake stock, so its alerts
// read the same levels the movements change.
type fakeReorder struct{ *store }

func (f fakeReorder) Save(_ context.Context, r *domain.ReorderRule) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v, ok := f.variants[r.VariantID()]; !ok || v.tenantID != r.TenantID() {
		return domain.ErrVariantNotFound
	}
	if l, ok := f.locations[r.LocationID()]; !ok || l.TenantID() != r.TenantID() {
		return domain.ErrLocationNotFound
	}
	f.rules[levelKey{r.TenantID(), r.VariantID(), r.LocationID()}] = r
	return nil
}

func (f fakeReorder) Delete(_ context.Context, tenantID, variantID, locationID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := levelKey{tenantID, variantID, locationID}
	if _, ok := f.rules[k]; !ok {
		return domain.ErrReorderRuleNotFound
	}
	delete(f.rules, k)
	return nil
}

func (f fakeReorder) ForVariants(_ context.Context, tenantID, locationID uuid.UUID, variantIDs []uuid.UUID) (map[uuid.UUID]*domain.ReorderRule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[uuid.UUID]*domain.ReorderRule{}
	for _, id := range variantIDs {
		if r, ok := f.rules[levelKey{tenantID, id, locationID}]; ok {
			out[id] = r
		}
	}
	return out, nil
}

// Alerts filters by the domain rule itself: the SQL condition is checked
// against it in the repository's integration tests.
func (f fakeReorder) Alerts(_ context.Context, tenantID uuid.UUID, filter domain.ReorderFilter) (domain.RuleStockPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reorderFilters = append(f.reorderFilters, filter)
	var all []domain.RuleStock
	for k, r := range f.rules {
		if k.tenantID != tenantID || filter.LocationID != nil && *filter.LocationID != k.locationID {
			continue
		}
		onHand := f.levels[k].Quantity
		if r.NeedsReorder(onHand) {
			all = append(all, domain.RuleStock{Rule: r, OnHand: onHand})
		}
	}
	slices.SortFunc(all, func(a, b domain.RuleStock) int {
		return a.OnHand.Sub(a.Rule.Min()).Cmp(b.OnHand.Sub(b.Rule.Min()))
	})
	page := domain.RuleStockPage{Total: len(all)}
	page.Items = all[min(filter.Offset, len(all)):min(filter.Offset+filter.Limit, len(all))]
	return page, nil
}
