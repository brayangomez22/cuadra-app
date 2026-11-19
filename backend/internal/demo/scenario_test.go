package demo_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/demo"
)

// demoTemplates are the demo profiles: one per business template, at least a
// hardware store and a clothing store (CLAUDE.md, "Neutralidad").
var demoTemplates = []string{"ferreteria", "ropa_calzado"}

// decimalThousand is a thousand pesos.
var decimalThousand = decimal.NewFromInt(1000)

// end is the last day of history of the small scenarios: from December to
// March, so they have a season and a quiet month.
var end = time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

// small is a scenario fast enough for every test.
func small(t *testing.T, template string, seed uint64) demo.Scenario {
	t.Helper()
	p, err := demo.LoadProfile(template)
	require.NoError(t, err)
	s, err := demo.NewScenario(p, demo.Options{Seed: seed, End: end, Days: 121, Products: 150, SalesPerDay: 20, Customers: 40})
	require.NoError(t, err)
	return s
}

func TestNewScenario(t *testing.T) {
	for _, template := range demoTemplates {
		t.Run("el mismo plan sale con la misma semilla: "+template, func(t *testing.T) {
			first, second := small(t, template, 7), small(t, template, 7)
			require.Equal(t, first, second)
		})

		t.Run("otra semilla da otro plan: "+template, func(t *testing.T) {
			require.NotEqual(t, small(t, template, 7).Products, small(t, template, 8).Products)
		})

		t.Run("el historial termina el día pedido y va en orden: "+template, func(t *testing.T) {
			s := small(t, template, 1)
			require.Len(t, s.Days, 121)
			require.Equal(t, end, s.Days[len(s.Days)-1].Date)
			var last time.Time
			for _, at := range eventTimes(s) {
				require.False(t, at.Before(last), "%s is before %s", at, last)
				last = at
			}
		})

		t.Run("el catálogo tiene el tamaño pedido: "+template, func(t *testing.T) {
			s := small(t, template, 1)
			require.Len(t, variants(s), 150)
		})
	}
}

func TestScenarioCalendar(t *testing.T) {
	for _, template := range demoTemplates {
		t.Run("los sábados se vende más que los martes: "+template, func(t *testing.T) {
			by := map[time.Weekday]int{}
			for _, d := range small(t, template, 3).Days {
				by[d.Date.Weekday()] += len(d.Sales)
			}
			require.Greater(t, by[time.Saturday], by[time.Tuesday])
		})

		t.Run("diciembre vende más que febrero: "+template, func(t *testing.T) {
			by := map[time.Month]int{}
			for _, d := range small(t, template, 3).Days {
				by[d.Date.Month()] += len(d.Sales)
			}
			// February has 28 days and December 31: compare per day.
			require.Greater(t, by[time.December]*28, by[time.February]*31)
		})
	}
}

func TestScenarioStock(t *testing.T) {
	for _, template := range demoTemplates {
		t.Run("ningún día vende más de lo que hay en stock, con lo que devuelven las anulaciones: "+template, func(t *testing.T) {
			s := small(t, template, 5)
			stock := map[[3]int]decimal.Decimal{}
			move := func(location int, lines []demo.Line, sign int64) {
				for _, l := range lines {
					k := [3]int{location, l.Product, l.Variant}
					stock[k] = stock[k].Add(l.Quantity.Mul(decimal.NewFromInt(sign)))
					require.False(t, stock[k].IsNegative(), "%s: %v leaves %s", template, k, stock[k])
				}
			}
			for _, d := range s.Days {
				for _, e := range d.Events() {
					switch {
					case e.Purchase != nil:
						move(e.Purchase.Location, e.Purchase.Lines, 1)
					case e.Sale != nil:
						move(e.Sale.Location, e.Sale.Lines, -1)
					case e.Void != nil:
						move(e.Void.Location, e.Void.Lines, 1)
					}
				}
			}
		})

		t.Run("se reponen compras durante el historial: "+template, func(t *testing.T) {
			purchases := 0
			for _, d := range small(t, template, 5).Days[1:] {
				purchases += len(d.Purchases)
			}
			require.Positive(t, purchases)
		})
	}
}
