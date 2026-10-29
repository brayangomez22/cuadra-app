package domain

import "github.com/shopspring/decimal"

// AgingBucket is how overdue a balance is.
type AgingBucket string

// Aging buckets: not yet due (due today or later), and 1–30, 31–60, 61–90
// and more than 90 days past the due date.
const (
	AgingCurrent    AgingBucket = "current"
	AgingDays1To30  AgingBucket = "days_1_30"
	AgingDays31To60 AgingBucket = "days_31_60"
	AgingDays61To90 AgingBucket = "days_61_90"
	AgingOver90     AgingBucket = "over_90"
)

// BucketOf returns the bucket of a balance due on due, seen on today.
func BucketOf(due, today Date) AgingBucket {
	switch days := today.DaysSince(due); {
	case days <= 0:
		return AgingCurrent
	case days <= 30:
		return AgingDays1To30
	case days <= 60:
		return AgingDays31To60
	case days <= 90:
		return AgingDays61To90
	default:
		return AgingOver90
	}
}

// Aging adds up balances by bucket, as of a day.
type Aging struct {
	today      Date
	Current    decimal.Decimal
	Days1To30  decimal.Decimal
	Days31To60 decimal.Decimal
	Days61To90 decimal.Decimal
	Over90     decimal.Decimal
}

// NewAging returns an empty aging as of today (in America/Bogota).
func NewAging(today Date) *Aging {
	return &Aging{today: today}
}

// Add adds a balance due on due to its bucket.
func (g *Aging) Add(due Date, balance decimal.Decimal) {
	switch BucketOf(due, g.today) {
	case AgingCurrent:
		g.Current = g.Current.Add(balance)
	case AgingDays1To30:
		g.Days1To30 = g.Days1To30.Add(balance)
	case AgingDays31To60:
		g.Days31To60 = g.Days31To60.Add(balance)
	case AgingDays61To90:
		g.Days61To90 = g.Days61To90.Add(balance)
	case AgingOver90:
		g.Over90 = g.Over90.Add(balance)
	}
}

// Overdue returns the balance past its due date.
func (g *Aging) Overdue() decimal.Decimal {
	return g.Days1To30.Add(g.Days31To60).Add(g.Days61To90).Add(g.Over90)
}

// Total returns the whole balance.
func (g *Aging) Total() decimal.Decimal { return g.Current.Add(g.Overdue()) }
