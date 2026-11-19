package domain

import (
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
)

// AgingLine is what a customer owes by age, for the receivables export.
type AgingLine struct {
	Name                                       string
	Current, Days1To30, Days31To60, Days61To90 decimal.Decimal
	Over90, Overdue, Total                     decimal.Decimal
}

// AgingExport is the whole aging as of a day, with its totals (Name empty).
type AgingExport struct {
	AsOf   calendar.Date
	Lines  []AgingLine
	Totals AgingLine
}
