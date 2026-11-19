package domain

import (
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
)

// CashFigures add up cash closings: how many, how many with a difference,
// what they sold, the shortages and overages (both positive) and the net
// difference (negative is short).
type CashFigures struct {
	Closings, WithDifference int
	SalesTotal               decimal.Decimal
	Shortage, Overage        decimal.Decimal
	Difference               decimal.Decimal
}

// Add returns the sum of f and o.
func (f CashFigures) Add(o CashFigures) CashFigures {
	return CashFigures{
		Closings: f.Closings + o.Closings, WithDifference: f.WithDifference + o.WithDifference,
		SalesTotal: f.SalesTotal.Add(o.SalesTotal), Shortage: f.Shortage.Add(o.Shortage),
		Overage: f.Overage.Add(o.Overage), Difference: f.Difference.Add(o.Difference),
	}
}

// Closing is a closed cash session: its cashier, its location, the day it
// closed in Colombia, what it sold and its difference (negative is short).
type Closing struct {
	UserID, LocationID     uuid.UUID
	UserName, LocationName string
	Day                    calendar.Date
	SalesTotal, Difference decimal.Decimal
}

// figures are what a single closing adds up to.
func (c Closing) figures() CashFigures {
	f := CashFigures{Closings: 1, SalesTotal: c.SalesTotal, Difference: c.Difference}
	if !c.Difference.IsZero() {
		f.WithDifference = 1
	}
	if c.Difference.IsNegative() {
		f.Shortage = c.Difference.Neg()
	} else {
		f.Overage = c.Difference
	}
	return f
}

// NewCashReport groups the closings by group, orders the groups (by day,
// or the most short first, then by name) and adds up the totals.
func NewCashReport(closings []Closing, group CashGroup) CashReport {
	r := CashReport{Group: group}
	index := map[string]int{}
	for _, c := range closings {
		key, label := c.groupKey(group)
		i, ok := index[key]
		if !ok {
			i = len(r.Rows)
			index[key] = i
			r.Rows = append(r.Rows, CashRow{Key: key, Label: label})
		}
		r.Rows[i].CashFigures = r.Rows[i].Add(c.figures())
	}
	slices.SortStableFunc(r.Rows, func(a, b CashRow) int {
		if group == CashByDay {
			return strings.Compare(a.Key, b.Key)
		}
		return mostFirst(a.Shortage, b.Shortage, a.Label, b.Label)
	})
	r.Sum()
	return r
}

func (c Closing) groupKey(group CashGroup) (key, label string) {
	switch group {
	case CashByLocation:
		return c.LocationID.String(), c.LocationName
	case CashByDay:
		return c.Day.String(), c.Day.String()
	default:
		return c.UserID.String(), c.UserName
	}
}

// CashRow is a group of closings: a user, a location or a day.
type CashRow struct {
	Key, Label string
	CashFigures
}

// CashReport is the period's cash closings by group.
type CashReport struct {
	Period Period
	Group  CashGroup
	Rows   []CashRow
	Totals CashFigures
}

// Sum sets the report's totals from its rows: every grouping splits the
// same closings.
func (r *CashReport) Sum() {
	r.Totals = CashFigures{}
	for _, row := range r.Rows {
		r.Totals = r.Totals.Add(row.CashFigures)
	}
}

// TeamFigures add up what a user did that the owner watches: voids,
// discounts, price changes, stock adjustments and cash closings with a
// difference.
type TeamFigures struct {
	Voids                           int
	VoidsTotal                      decimal.Decimal
	Discounts                       int
	DiscountsAmount, DiscountsGross decimal.Decimal
	PriceChanges, Adjustments       int
	AdjustmentsValue                decimal.Decimal
	CashDifferences                 int
	CashShortage, CashOverage       decimal.Decimal
}

// Add returns the sum of f and o.
func (f TeamFigures) Add(o TeamFigures) TeamFigures {
	return TeamFigures{
		Voids: f.Voids + o.Voids, VoidsTotal: f.VoidsTotal.Add(o.VoidsTotal),
		Discounts: f.Discounts + o.Discounts, DiscountsAmount: f.DiscountsAmount.Add(o.DiscountsAmount),
		DiscountsGross: f.DiscountsGross.Add(o.DiscountsGross), PriceChanges: f.PriceChanges + o.PriceChanges,
		Adjustments: f.Adjustments + o.Adjustments, AdjustmentsValue: f.AdjustmentsValue.Add(o.AdjustmentsValue),
		CashDifferences: f.CashDifferences + o.CashDifferences, CashShortage: f.CashShortage.Add(o.CashShortage),
		CashOverage: f.CashOverage.Add(o.CashOverage),
	}
}

// DiscountRate is what was discounted over the value of the discounted
// sales; false without discounts.
func (f TeamFigures) DiscountRate() (decimal.Decimal, bool) {
	if f.Discounts == 0 {
		return decimal.Zero, false
	}
	return Rate(f.DiscountsAmount, f.DiscountsGross)
}

// TeamPoint is what a user did in a bucket of the trend.
type TeamPoint struct {
	UserID      uuid.UUID
	Name        string
	BucketStart calendar.Date
	TeamFigures
}

// TeamUser is what a user did in the whole period.
type TeamUser struct {
	UserID uuid.UUID
	Name   string
	TeamFigures
}

// TeamReport is the team control of the period: by user and as a trend.
type TeamReport struct {
	Period Period
	Bucket Bucket
	Users  []TeamUser
	Series []TeamPoint
	Costs  bool
}

// SumByUser adds up the points of each user, ordered by name (then id).
func SumByUser(points []TeamPoint) []TeamUser {
	index := map[uuid.UUID]int{}
	var users []TeamUser
	for _, p := range points {
		i, ok := index[p.UserID]
		if !ok {
			i = len(users)
			index[p.UserID] = i
			users = append(users, TeamUser{UserID: p.UserID, Name: p.Name})
		}
		users[i].TeamFigures = users[i].Add(p.TeamFigures)
	}
	slices.SortFunc(users, func(a, b TeamUser) int {
		if c := strings.Compare(a.Name, b.Name); c != 0 {
			return c
		}
		return strings.Compare(a.UserID.String(), b.UserID.String())
	})
	return users
}

// HideCosts zeroes the adjustments' value, for a caller who may not see
// costs.
func (r *TeamReport) HideCosts() {
	r.Costs = false
	for i := range r.Users {
		r.Users[i].AdjustmentsValue = decimal.Zero
	}
	for i := range r.Series {
		r.Series[i].AdjustmentsValue = decimal.Zero
	}
}
