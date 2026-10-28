package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// moneyScale is the decimals of a value in pesos.
const moneyScale = 2

// CountLine is a variant in a stock count: its stock and average cost when
// the count started (the snapshot) and its round, which goes up with each
// recount.
type CountLine struct {
	VariantID     uuid.UUID
	StartQuantity decimal.Decimal
	StartCost     decimal.Decimal
	Round         int
}

// CounterTotal is how much one person counted of a variant in the current
// round, adding up their entries.
type CounterTotal struct {
	UserID   uuid.UUID
	Quantity decimal.Decimal
}

// SheetLine is a line with what is known of it to evaluate it: what each
// person counted in its current round and the net quantity of the kardex
// movements of the variant at the location from the start of the count to
// its last entry (sales and receipts while counting).
type SheetLine struct {
	CountLine
	Counts    []CounterTotal
	Movements decimal.Decimal
}

// LineStatus is the outcome of a line.
type LineStatus string

// Line statuses: nobody counted it; it matches; it differs; or, in review
// mode, the counters disagree and it must be recounted.
const (
	LineUncounted  LineStatus = "uncounted"
	LineExact      LineStatus = "exact"
	LineDifference LineStatus = "difference"
	LineReview     LineStatus = "review"
)

// ParseLineStatus returns the status with code s. Matching is exact.
func ParseLineStatus(s string) (LineStatus, error) {
	switch st := LineStatus(s); st {
	case LineUncounted, LineExact, LineDifference, LineReview:
		return st, nil
	}
	return "", ErrInvalidLineStatus
}

// LineResult is an evaluated line. Expected is the stock the variant should
// have had when it was counted; Difference is counted − expected and Value
// that difference at the starting cost, in pesos. Counted, Difference and
// Value are zero for an uncounted line or one under review.
type LineResult struct {
	VariantID  uuid.UUID
	Status     LineStatus
	Expected   decimal.Decimal
	Counted    decimal.Decimal
	Difference decimal.Decimal
	Value      decimal.Decimal
}

// EvaluateLine compares what was counted of a line with what was expected:
// the stock at the start plus the inbound minus the outbound movements while
// counting. The business keeps selling during a count, so the snapshot alone
// would turn every sale into a missing unit. The value is rounded half-up to
// two decimals.
func EvaluateLine(l SheetLine, mode CountMode) LineResult {
	r := LineResult{VariantID: l.VariantID, Expected: l.StartQuantity.Add(l.Movements)}
	if len(l.Counts) == 0 {
		r.Status = LineUncounted
		return r
	}
	counted, agreed := l.counted(mode)
	if !agreed {
		r.Status = LineReview
		return r
	}
	r.Counted = counted
	r.Difference = counted.Sub(r.Expected)
	r.Value = r.Difference.Mul(l.StartCost).Round(moneyScale)
	r.Status = LineExact
	if !r.Difference.IsZero() {
		r.Status = LineDifference
	}
	return r
}

// counted combines the counts of the line, which has at least one: their sum
// in sum mode; in review mode the quantity they all agree on, or false if
// they disagree.
func (l SheetLine) counted(mode CountMode) (decimal.Decimal, bool) {
	first := l.Counts[0].Quantity
	total := decimal.Zero
	for _, c := range l.Counts {
		if mode == CountReview && !c.Quantity.Equal(first) {
			return decimal.Zero, false
		}
		total = total.Add(c.Quantity)
	}
	if mode == CountReview {
		return first, true
	}
	return total, true
}

// CountSummary is the accuracy of a count: how many lines it had, how many
// were counted and matched, and the value of the differences (net, and the
// sum of their absolute values).
type CountSummary struct {
	Lines         int
	Counted       int
	Exact         int
	NetValue      decimal.Decimal
	AbsoluteValue decimal.Decimal
}

// Summarize adds up the results of a count's lines.
func Summarize(results []LineResult) CountSummary {
	s := CountSummary{Lines: len(results), NetValue: decimal.Zero, AbsoluteValue: decimal.Zero}
	for _, r := range results {
		switch r.Status {
		case LineExact:
			s.Exact++
			s.Counted++
		case LineDifference:
			s.Counted++
			s.NetValue = s.NetValue.Add(r.Value)
			s.AbsoluteValue = s.AbsoluteValue.Add(r.Value.Abs())
		}
	}
	return s
}

// Accuracy is the share of counted lines that matched, from 0 to 1 with
// four decimals; false if no line was counted.
func (s CountSummary) Accuracy() (decimal.Decimal, bool) {
	if s.Counted == 0 {
		return decimal.Zero, false
	}
	return decimal.NewFromInt(int64(s.Exact)).DivRound(decimal.NewFromInt(int64(s.Counted)), amountScale), true
}
