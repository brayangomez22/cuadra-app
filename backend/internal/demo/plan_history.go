package demo

import (
	"math/rand/v2"
	"time"

	"github.com/shopspring/decimal"
)

// The business hours, in Bogotá: payments come in before the first sale,
// so they never pay a debt the day's sales have not made yet.
const (
	openingHour  = 8
	firstSaleMin = 9 * 60
	lastSaleMin  = 19 * 60
	// otherLocationShare is how much the other locations sell compared
	// with the first, in percent.
	otherLocationShare = 70
	// creditMargin keeps credit sales this percent under the limit: the
	// API's total may differ from Total by some cents.
	creditMargin = 95
	// payOneIn is the chance (one in) that a customer who owes pays on a
	// given day.
	payOneIn = 8
)

// stockKey is a variant at a location.
type stockKey struct{ location, product, variant int }

// history plans the days in order, keeping what the business has and owes
// so no sale takes what is not in stock and no credit passes its limit.
type history struct {
	rng      *rand.Rand
	s        *Scenario
	o        Options
	start    time.Time
	stock    map[stockKey]decimal.Decimal
	level    map[stockKey]decimal.Decimal
	owed     []decimal.Decimal
	popular  []int
	credit   []int
	invoices int
}

func newHistory(rng *rand.Rand, s *Scenario, o Options) *history {
	h := &history{
		rng:     rng,
		s:       s,
		o:       o,
		start:   o.End.AddDate(0, 0, 1-o.Days),
		stock:   map[stockKey]decimal.Decimal{},
		level:   map[stockKey]decimal.Decimal{},
		owed:    make([]decimal.Decimal, len(s.Customers)),
		popular: rng.Perm(len(s.Products)),
	}
	h.credit = creditCustomers(h)
	return h
}

// plan returns the days: the first one receives the initial stock; then
// each supplier restocks once a week what runs low.
func (h *history) plan() []Day {
	days := make([]Day, h.o.Days)
	for i := range days {
		date := h.start.AddDate(0, 0, i)
		d := Day{Date: date}
		if i == 0 {
			d.Purchases = h.initialStock(date)
		} else {
			d.Purchases = h.restock(date)
		}
		d.Payments = h.payments(date)
		d.Sales = h.sales(date)
		days[i] = d
	}
	return days
}

// at is the time minutes after midnight of date, in Bogotá.
func at(date time.Time, minutes int) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, minutes, 0, 0, bogota)
}

// payments are the customers with credit (not the defaulters) paying part
// of what they owe, early in the morning.
func (h *history) payments(date time.Time) []Payment {
	var out []Payment
	for i, c := range h.s.Customers {
		if !c.CreditLimit.IsPositive() || c.Defaulter || h.owed[i].LessThan(decimal.NewFromInt(2000)) || h.rng.IntN(payOneIn) != 0 {
			continue
		}
		share := decimal.NewFromInt(int64(40 + h.rng.IntN(61)))
		// Whole thousands, and a peso under the debt so cents never make it
		// more than what the API says is owed.
		amount := h.owed[i].Sub(decimal.NewFromInt(1)).Mul(share).Div(decimal.NewFromInt(100)).
			Div(decimal.NewFromInt(1000)).Floor().Mul(decimal.NewFromInt(1000))
		if !amount.IsPositive() {
			continue
		}
		h.owed[i] = h.owed[i].Sub(amount)
		method := "cash"
		if h.rng.IntN(5) < 2 {
			method = "transfer"
		}
		out = append(out, Payment{At: at(date, openingHour*60+10+len(out)), Customer: i, Amount: amount, Method: method})
	}
	return out
}

// salesToday is how many sales a location makes on date: the normal day
// by the weekday and the month, give or take 15 %.
func (h *history) salesToday(date time.Time, location int) int {
	n := h.o.SalesPerDay * h.s.Profile.Weekdays[date.Weekday()] * h.s.Profile.Months[date.Month()-1]
	n = n * (85 + h.rng.IntN(31)) / 100
	if location > 0 {
		n = n * otherLocationShare / 100
	}
	return n / 10_000
}
