package domain

import "github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"

// Date is the shared kernel's calendar day, in America/Bogota: the invoice
// date and the due date of a purchase.
type Date = calendar.Date

// The shared kernel's constructors of a Date.
var (
	NewDate = calendar.NewDate
	DateIn  = calendar.DateIn
	DateOf  = calendar.DateOf
)
