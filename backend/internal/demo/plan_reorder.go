package demo

import "github.com/shopspring/decimal"

// reorderShare is the part of the catalog, the best sellers, with a minimum
// and maximum stock: one product in reorderShare. minimumDivisor puts the
// minimum at a fourth of the level, under where the plan restocks (a third).
const (
	reorderShare   = 4
	minimumDivisor = 4
)

// reorderRules gives the best-selling articles of each location a minimum
// stock at a fourth of their level and a maximum at their level. The weekly
// restock comes at a third, so only what sold fast between visits reaches
// its minimum: the demo shows some tens of reorder alerts per location, not
// a hundred. It draws nothing from rng: the rest of the scenario does not change
// with it.
func (h *history) reorderRules() []ReorderRule {
	best := h.popular[:max(1, len(h.popular)/reorderShare)]
	var out []ReorderRule
	for loc := range h.s.Locations {
		for _, p := range best {
			for v := range h.s.Products[p].Variants {
				level := h.level[stockKey{loc, p, v}]
				low := level.Div(decimal.NewFromInt(minimumDivisor)).Floor()
				if !low.IsPositive() {
					continue
				}
				out = append(out, ReorderRule{Location: loc, Product: p, Variant: v, Min: low, Max: level})
			}
		}
	}
	return out
}
