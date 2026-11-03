package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// dueSoonDays is how far ahead "due soon" looks: what to plan paying this
// week.
const dueSoonDays = 7

// PayablesWindow splits what is owed by due date: before OverdueBefore it
// is overdue; up to DueSoonUntil (included) it falls due soon.
type PayablesWindow struct {
	OverdueBefore Date
	DueSoonUntil  Date
}

// PayablesWindowOn is the window of a day in America/Bogota.
func PayablesWindowOn(today Date) PayablesWindow {
	return PayablesWindow{OverdueBefore: today, DueSoonUntil: today.AddDays(dueSoonDays)}
}

// SupplierPayable is what the tenant owes a supplier: the sum of its open
// purchases, how much of it is overdue or falls due soon, and the earliest
// due date.
type SupplierPayable struct {
	SupplierID   uuid.UUID
	SupplierName string
	Balance      decimal.Decimal
	Overdue      decimal.Decimal
	DueSoon      decimal.Decimal
	Purchases    int
	NextDue      Date
}

// PayablesTotals adds up what the tenant owes every supplier.
type PayablesTotals struct {
	Balance   decimal.Decimal
	Overdue   decimal.Decimal
	DueSoon   decimal.Decimal
	Suppliers int
}
