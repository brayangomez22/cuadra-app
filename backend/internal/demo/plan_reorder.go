package demo

import "github.com/shopspring/decimal"

// reorderShare is the part of the catalog, the best sellers, with a minimum
// and maximum stock: one product in reorderShare.
const reorderShare = 4

// reorderRules gives the best-selling articles of each location a minimum
// stock at the level the plan restocks them (a third of their level) and a
// maximum at their level. The weekly restock lets the best sellers go under
// their minimum between visits, so the demo shows reorder alerts. It draws
// nothing from rng: the rest of the scenario does not change with it.
func (h *history) reorderRules() []ReorderRule {
	best := h.popular[:max(1, len(h.popular)/reorderShare)]
	var out []ReorderRule
	for loc := range h.s.Locations {
		for _, p := range best {
			for v := range h.s.Products[p].Variants {
				level := h.level[stockKey{loc, p, v}]
				low := level.Div(decimal.NewFromInt(restockDivisor)).Floor()
				if !low.IsPositive() {
					continue
				}
				out = append(out, ReorderRule{Location: loc, Product: p, Variant: v, Min: low, Max: level})
			}
		}
	}
	return out
}
