package demo

import (
	"time"

	"github.com/shopspring/decimal"
)

// The team's control, so the dashboard has something to show: the
// administrator voids a sale on one day in voidOneIn, and in today's week
// (the one the dashboard watches) unusualVoids more spread over its days;
// the cashier closes that week's last drawer unusualShortage short. A void
// is voidAfter to voidAfter+voidSlack minutes after the sale, so it is
// always before closing.
const (
	voidOneIn       = 4
	unusualVoids    = 8
	unusualShortage = 70_000
	voidAfter       = 5
	voidSlack       = 10
)

// voidReasons are why the administrator voids a sale.
var voidReasons = []string{
	"El cliente desistió de la compra",
	"Se registró dos veces",
	"Se cobró un producto equivocado",
	"El pago no se completó",
}

// watched are the days of history in today's week (today is the day after
// End): none when today is a Monday.
func (h *history) watched() (from time.Time, days int) {
	today := h.o.End.AddDate(0, 0, 1)
	offset := (int(today.Weekday()) + 6) % 7
	return today.AddDate(0, 0, -offset), offset
}

// unusualVoidsOn is how many of the week's unusual voids fall on date: an
// even share of them on each watched day.
func (h *history) unusualVoidsOn(date time.Time) int {
	from, days := h.watched()
	if days == 0 || date.Before(from) {
		return 0
	}
	k := int(date.Sub(from).Hours() / 24)
	return unusualVoids*(k+1)/days - unusualVoids*k/days
}

// shortCashier leaves the cashier's drawer (the first location's)
// unusualShortage short on the last day of today's week.
func (h *history) shortCashier(d *Day) {
	if _, days := h.watched(); days > 0 && d.Date.Equal(h.o.End) {
		d.CashDifference[0] = decimal.NewFromInt(-unusualShortage)
	}
}

// voids voids some of the day's sales paid by card or transfer (cash would
// change what the drawer should have, credit what the customer owes) and
// gives their stock back.
func (h *history) voids(d *Day) {
	n := h.unusualVoidsOn(d.Date)
	if h.rng.IntN(voidOneIn) == 0 {
		n++
	}
	var voidable []int
	for i, s := range d.Sales {
		if s.Method != "cash" && !s.Credit {
			voidable = append(voidable, i)
		}
	}
	h.rng.Shuffle(len(voidable), func(i, j int) { voidable[i], voidable[j] = voidable[j], voidable[i] })
	for _, i := range voidable[:min(n, len(voidable))] {
		s := &d.Sales[i]
		after := time.Duration(voidAfter+h.rng.IntN(voidSlack+1)) * time.Minute
		s.Void = &Void{At: s.At.Add(after), Reason: voidReasons[h.rng.IntN(len(voidReasons))]}
		for _, l := range s.Lines {
			k := stockKey{s.Location, l.Product, l.Variant}
			h.stock[k] = h.stock[k].Add(l.Quantity)
		}
	}
}
