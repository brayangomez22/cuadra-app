package demo_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/demo"
)

// key is a variant at a location of a scenario.
type key struct{ location, product, variant int }

// finalStock replays the scenario's purchases and sales: what each variant
// has at each location when the history ends.
func finalStock(s demo.Scenario) map[key]decimal.Decimal {
	stock := map[key]decimal.Decimal{}
	for _, d := range s.Days {
		for _, p := range d.Purchases {
			for _, l := range p.Lines {
				k := key{p.Location, l.Product, l.Variant}
				stock[k] = stock[k].Add(l.Quantity)
			}
		}
		for _, sale := range d.Sales {
			for _, l := range sale.Lines {
				k := key{sale.Location, l.Product, l.Variant}
				stock[k] = stock[k].Sub(l.Quantity)
			}
		}
	}
	return stock
}

func TestScenarioReorder(t *testing.T) {
	for _, template := range demoTemplates {
		t.Run("siembra stock mínimo y máximo para parte de las variantes de cada sede: "+template, func(t *testing.T) {
			s := small(t, template, 1)
			require.NotEmpty(t, s.Reorder)
			require.Less(t, len(s.Reorder), len(variants(s))*len(s.Locations))

			byLocation := map[int]int{}
			seen := map[key]bool{}
			for _, r := range s.Reorder {
				k := key{r.Location, r.Product, r.Variant}
				require.False(t, seen[k], "one rule per variant and location")
				seen[k] = true
				byLocation[r.Location]++
				require.True(t, r.Min.IsPositive(), "min %s", r.Min)
				require.True(t, r.Min.LessThan(r.Max), "min %s, max %s", r.Min, r.Max)
				require.True(t, r.Min.Equal(r.Min.Floor()), "whole quantities")
			}
			require.Len(t, byLocation, len(s.Locations))
		})

		t.Run("al final del historial hay variantes bajo el mínimo, y no todas: "+template, func(t *testing.T) {
			s := small(t, template, 1)
			stock := finalStock(s)
			below := 0
			for _, r := range s.Reorder {
				if stock[key{r.Location, r.Product, r.Variant}].LessThan(r.Min) {
					below++
				}
			}
			require.Positive(t, below)
			require.Less(t, below, len(s.Reorder))
		})
	}
}
