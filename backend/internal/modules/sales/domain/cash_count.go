package domain

import (
	"slices"

	"github.com/shopspring/decimal"
)

// maxDenominationQuantity bounds the bills or coins of one denomination, so
// a typo cannot overflow the total.
const maxDenominationQuantity = 10_000_000

// denominations are the bills and coins of the Colombian peso in circulation,
// from the largest. The 1000 bill and coin count together.
var denominations = []decimal.Decimal{
	decimal.NewFromInt(100_000), decimal.NewFromInt(50_000), decimal.NewFromInt(20_000),
	decimal.NewFromInt(10_000), decimal.NewFromInt(5_000), decimal.NewFromInt(2_000),
	decimal.NewFromInt(1_000), decimal.NewFromInt(500), decimal.NewFromInt(200),
	decimal.NewFromInt(100), decimal.NewFromInt(50),
}

// Denominations returns the bills and coins a count accepts, from the
// largest.
func Denominations() []decimal.Decimal { return slices.Clone(denominations) }

func isDenomination(d decimal.Decimal) bool {
	return slices.ContainsFunc(denominations, d.Equal)
}

// DenominationCount is how many bills or coins of a denomination were
// counted.
type DenominationCount struct {
	Denomination decimal.Decimal
	Quantity     int64
}

// CashCount is the count of a drawer by denomination (arqueo): each
// denomination once, from the largest.
type CashCount struct {
	counts []DenominationCount
}

// NewCashCount checks a count: denominations of the peso in circulation,
// each at most once, with a quantity from 0 to ten million. An empty count is
// an empty drawer.
func NewCashCount(counts []DenominationCount) (CashCount, error) {
	out := make([]DenominationCount, 0, len(counts))
	for _, c := range counts {
		if !isDenomination(c.Denomination) {
			return CashCount{}, ErrInvalidDenomination
		}
		if c.Quantity < 0 || c.Quantity > maxDenominationQuantity {
			return CashCount{}, ErrInvalidDenominationQuantity
		}
		if slices.ContainsFunc(out, func(o DenominationCount) bool { return o.Denomination.Equal(c.Denomination) }) {
			return CashCount{}, ErrDuplicateDenomination
		}
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b DenominationCount) int { return b.Denomination.Cmp(a.Denomination) })
	return CashCount{counts: out}, nil
}

// Total returns the cash counted.
func (c CashCount) Total() decimal.Decimal {
	total := decimal.Zero
	for _, d := range c.counts {
		total = total.Add(d.Denomination.Mul(decimal.NewFromInt(d.Quantity)))
	}
	return total
}

// Counts returns a copy of the count, from the largest denomination.
func (c CashCount) Counts() []DenominationCount { return slices.Clone(c.counts) }
