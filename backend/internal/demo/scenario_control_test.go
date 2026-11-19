package demo_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/demo"
)

// watchedWeeks is how many weeks, with today's, the dashboard measures the
// team's normal on.
const watchedWeeks = 8

// mondayOf is the Monday of the week of day (a date at midnight UTC).
func mondayOf(day time.Time) time.Time {
	return day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
}

// weekly adds up, by the Monday of each of the watched weeks (today's
// included: the day after the scenario's end), what figure gives each day.
func weekly(s demo.Scenario, figure func(demo.Day) decimal.Decimal) (today decimal.Decimal, mean decimal.Decimal) {
	thisWeek := mondayOf(s.Days[len(s.Days)-1].Date.AddDate(0, 0, 1))
	from := thisWeek.AddDate(0, 0, -7*(watchedWeeks-1))
	total := decimal.Zero
	for _, d := range s.Days {
		if d.Date.Before(from) {
			continue
		}
		v := figure(d)
		total = total.Add(v)
		if !d.Date.Before(thisWeek) {
			today = today.Add(v)
		}
	}
	return today, total.Div(decimal.NewFromInt(watchedWeeks))
}

// voids is how many sales of the day are voided.
func voids(d demo.Day) decimal.Decimal {
	n := 0
	for _, s := range d.Sales {
		if s.Void != nil {
			n++
		}
	}
	return decimal.NewFromInt(int64(n))
}

// shortage is how much the first location's drawer (the cashier's) closed short.
func shortage(d demo.Day) decimal.Decimal {
	return decimal.Min(d.CashDifference[0], decimal.Zero).Neg()
}

// ending is a small scenario whose history ends on the given day.
func ending(t *testing.T, template string, last time.Time) demo.Scenario {
	t.Helper()
	p, err := demo.LoadProfile(template)
	require.NoError(t, err)
	s, err := demo.NewScenario(p, demo.Options{Seed: 4, End: last, Days: 121, Products: 150, SalesPerDay: 20, Customers: 40})
	require.NoError(t, err)
	return s
}

func TestScenarioTeamControl(t *testing.T) {
	for _, template := range demoTemplates {
		t.Run("se anulan algunas ventas, solo pagadas sin efectivo ni crédito, minutos después y antes del cierre: "+template, func(t *testing.T) {
			s := small(t, template, 2)
			voided := 0
			for _, d := range s.Days {
				for _, sale := range d.Sales {
					if sale.Void == nil {
						continue
					}
					voided++
					require.NotEqual(t, "cash", sale.Method)
					require.False(t, sale.Credit)
					require.NotEmpty(t, sale.Void.Reason)
					require.True(t, sale.Void.At.After(sale.At), "voided at %s, sold at %s", sale.Void.At, sale.At)
					require.True(t, sale.Void.At.Before(d.Closing), "voided at %s, closed at %s", sale.Void.At, d.Closing)
				}
			}
			require.Greater(t, voided, 10)
		})

		t.Run("en la semana de hoy se anula más del doble que en una semana normal: "+template, func(t *testing.T) {
			today, mean := weekly(small(t, template, 2), voids)
			require.True(t, today.GreaterThan(mean.Mul(decimal.NewFromInt(2))), "this week %s, mean %s", today, mean)
		})

		t.Run("en la semana de hoy el cajero cierra con un faltante de más del doble del normal: "+template, func(t *testing.T) {
			today, mean := weekly(small(t, template, 2), shortage)
			require.True(t, today.GreaterThan(mean.Mul(decimal.NewFromInt(2))), "this week %s, mean %s", today, mean)
		})

		t.Run("si hoy es lunes la semana no tiene historia y nada sale fuera de lo normal: "+template, func(t *testing.T) {
			sunday := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
			s := ending(t, template, sunday)
			for _, d := range s.Days {
				require.True(t, shortage(d).LessThanOrEqual(decimal.NewFromInt(20_000)), "%s short %s", d.Date, shortage(d))
				require.LessOrEqual(t, voids(d).IntPart(), int64(1), "%s", d.Date)
			}
		})
	}
}
