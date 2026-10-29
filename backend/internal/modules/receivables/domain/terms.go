package domain

import "github.com/shopspring/decimal"

// maxTermDays is the longest credit term: a year.
const maxTermDays = 365

// Terms are the credit a tenant gives a customer: the most it may owe
// (zero: no credit) and how many days it has to pay each sale.
type Terms struct {
	limit decimal.Decimal
	days  int
}

// NewTerms returns the terms of a limit of COP (zero or more, at most two
// decimals) and a term of 0 to 365 days.
func NewTerms(limit decimal.Decimal, days int) (Terms, error) {
	if limit.IsNegative() || !isMoney(limit) {
		return Terms{}, ErrInvalidCreditLimit
	}
	if days < 0 || days > maxTermDays {
		return Terms{}, ErrInvalidCreditTerm
	}
	return Terms{limit: limit, days: days}, nil
}

// Limit returns the most the customer may owe.
func (t Terms) Limit() decimal.Decimal { return t.limit }

// Days returns the days the customer has to pay a sale.
func (t Terms) Days() int { return t.days }

// Enabled reports whether the terms give any credit.
func (t Terms) Enabled() bool { return t.limit.IsPositive() }
