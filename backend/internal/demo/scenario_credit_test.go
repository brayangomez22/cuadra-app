package demo_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/demo"
)

func TestScenarioCredit(t *testing.T) {
	for _, template := range demoTemplates {
		t.Run("ninguna venta a crédito pasa el cupo y ningún abono pasa la deuda: "+template, func(t *testing.T) {
			s := small(t, template, 4)
			owed := make([]decimal.Decimal, len(s.Customers))
			credit := 0
			for _, d := range s.Days {
				for _, e := range d.Events() {
					switch {
					case e.Sale != nil && e.Sale.Credit:
						credit++
						c := s.Customers[e.Sale.Customer]
						owed[e.Sale.Customer] = owed[e.Sale.Customer].Add(s.Total(*e.Sale))
						require.True(t, c.CreditLimit.IsPositive(), "a credit sale needs a customer with credit")
						require.False(t, owed[e.Sale.Customer].GreaterThan(c.CreditLimit),
							"%s owes %s over a limit of %s", c.Name, owed[e.Sale.Customer], c.CreditLimit)
					case e.Payment != nil:
						owed[e.Payment.Customer] = owed[e.Payment.Customer].Sub(e.Payment.Amount)
						require.False(t, owed[e.Payment.Customer].IsNegative(), "a payment over the debt")
					}
				}
			}
			require.Positive(t, credit)
		})

		t.Run("queda cartera vencida para el informe de edades: "+template, func(t *testing.T) {
			s := small(t, template, 4)
			last := s.Days[len(s.Days)-1].Date
			overdue := false
			for _, d := range s.Days {
				for _, sale := range d.Sales {
					if sale.Credit && last.Sub(d.Date) > time.Duration(s.Customers[sale.Customer].TermDays)*24*time.Hour {
						overdue = true
					}
				}
			}
			require.True(t, overdue)
		})

		t.Run("los clientes tienen documento único y algunos tienen cupo: "+template, func(t *testing.T) {
			s := small(t, template, 4)
			require.Len(t, s.Customers, 40)
			docs, withCredit := map[string]bool{}, 0
			for _, c := range s.Customers {
				key := c.DocumentType + c.Document
				require.False(t, docs[key], key)
				docs[key] = true
				if c.CreditLimit.IsPositive() {
					withCredit++
				}
			}
			require.Positive(t, withCredit)
			require.Less(t, withCredit, len(s.Customers))
		})
	}
}

func TestScenarioFullScale(t *testing.T) {
	cases := []struct {
		template string
		products int
	}{
		{"ferreteria", 3000},
		{"ropa_calzado", 600},
	}
	for _, c := range cases {
		t.Run("a escala completa tiene unos 3.000 artículos y 6 meses de ventas: "+c.template, func(t *testing.T) {
			p, err := demo.LoadProfile(c.template)
			require.NoError(t, err)
			s, err := demo.NewScenario(p, demo.Options{Seed: 1, End: end})
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(s.Products), c.products)
			require.GreaterOrEqual(t, len(variants(s)), 3000)
			require.Len(t, s.Days, demo.HistoryDays)
			require.Len(t, s.Locations, 2)
		})
	}
}
