package demo

import (
	"slices"
	"time"

	"github.com/shopspring/decimal"
)

// The shape of a sale: up to maxLines lines, tries to find each one in
// stock, and a buyer one in customerOneIn times (half of them with credit).
const (
	maxLines      = 4
	lineTries     = 6
	customerOneIn = 3
)

// methods are the payment methods of the counter, with their weight in
// percent.
var methods = []struct {
	name   string
	weight int
}{{"cash", 55}, {"card", 15}, {"nequi", 15}, {"daviplata", 5}, {"transfer", 10}}

// sales are the day's sales of every location, in time order.
func (h *history) sales(date time.Time) []Sale {
	var out []Sale
	for loc := range h.s.Locations {
		n := h.salesToday(date, loc)
		minutes := make([]int, n)
		for i := range minutes {
			minutes[i] = firstSaleMin + h.rng.IntN(lastSaleMin-firstSaleMin)
		}
		slices.Sort(minutes)
		for _, m := range minutes {
			if sale, ok := h.sale(at(date, m), loc); ok {
				out = append(out, sale)
			}
		}
	}
	slices.SortStableFunc(out, func(a, b Sale) int { return a.At.Compare(b.At) })
	return out
}

// sale takes up to maxLines articles that are in stock at loc; ok is false
// when it found none.
func (h *history) sale(t time.Time, loc int) (Sale, bool) {
	s := Sale{At: t, Location: loc, Customer: -1}
	lines := 1 + h.rng.IntN(maxLines)
	for range lines {
		if l, ok := h.line(loc, s.Lines); ok {
			s.Lines = append(s.Lines, l)
		}
	}
	if len(s.Lines) == 0 {
		return Sale{}, false
	}
	h.charge(&s)
	return s, true
}

// line picks a popular article with stock for the quantity it takes and
// takes it out of the stock.
func (h *history) line(loc int, taken []Line) (Line, bool) {
	for range lineTries {
		// The lower of two draws: the first products of popular sell most.
		p := h.popular[min(h.rng.IntN(len(h.popular)), h.rng.IntN(len(h.popular)))]
		v := h.rng.IntN(len(h.s.Products[p].Variants))
		if slices.ContainsFunc(taken, func(l Line) bool { return l.Product == p && l.Variant == v }) {
			continue
		}
		q := h.quantity(h.s.Products[p])
		k := stockKey{loc, p, v}
		if h.stock[k].LessThan(q) {
			continue
		}
		h.stock[k] = h.stock[k].Sub(q)
		return Line{Product: p, Variant: v, Quantity: q}, true
	}
	return Line{}, false
}

// quantity is how much a line takes, mostly little: in units, or in steps
// for what is sold by fractions.
func (h *history) quantity(p Product) decimal.Decimal {
	steps := p.MaxQuantity
	if !p.Step.Equal(decimal.NewFromInt(1)) {
		steps = int(decimal.NewFromInt(int64(p.MaxQuantity)).Div(p.Step).IntPart())
	}
	n := 1 + h.rng.IntN(1+h.rng.IntN(steps))
	return p.Step.Mul(decimal.NewFromInt(int64(n)))
}

// charge chooses who buys and how they pay: on credit only a customer with
// room in their limit.
func (h *history) charge(s *Sale) {
	s.Method = h.method()
	if h.rng.IntN(customerOneIn) != 0 {
		return
	}
	credit := h.credit
	if len(credit) > 0 && h.rng.IntN(2) == 0 {
		s.Customer = credit[h.rng.IntN(len(credit))]
		total := h.s.Total(*s)
		limit := h.creditRoom(h.s.Customers[s.Customer])
		if h.rng.IntN(2) == 0 && !h.owed[s.Customer].Add(total).GreaterThan(limit) {
			s.Credit, s.Method = true, "credit"
			h.owed[s.Customer] = h.owed[s.Customer].Add(total)
		}
		return
	}
	s.Customer = h.rng.IntN(len(h.s.Customers))
}

// creditRoom is the most a customer may owe: their limit with a margin, or
// half of it for one who never pays, because the shop stops giving credit
// to whoever does not pay.
func (h *history) creditRoom(c Customer) decimal.Decimal {
	share := int64(creditMargin)
	if c.Defaulter {
		share = defaulterShare
	}
	return c.CreditLimit.Mul(decimal.NewFromInt(share)).Div(decimal.NewFromInt(100))
}

// creditCustomers are the indexes of the customers with credit.
func creditCustomers(h *history) []int {
	var out []int
	for i, c := range h.s.Customers {
		if c.CreditLimit.IsPositive() {
			out = append(out, i)
		}
	}
	return out
}

// method is a payment method by its weight.
func (h *history) method() string {
	r := h.rng.IntN(100)
	for _, m := range methods {
		if r < m.weight {
			return m.name
		}
		r -= m.weight
	}
	return methods[0].name
}
