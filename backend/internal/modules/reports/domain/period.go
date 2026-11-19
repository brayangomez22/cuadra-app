package domain

import (
	"time"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
)

// MaxPeriodDays is the longest period of a report: a year, leap or not.
const MaxPeriodDays = 366

// Period is the days a report covers in Colombia, both included.
type Period struct{ from, to calendar.Date }

// NewPeriod returns the period from..to, or ErrInvalidPeriod if to is
// before from or it spans more than MaxPeriodDays.
func NewPeriod(from, to calendar.Date) (Period, error) {
	if to.Before(from) || to.DaysSince(from) >= MaxPeriodDays {
		return Period{}, ErrInvalidPeriod
	}
	return Period{from: from, to: to}, nil
}

// From is the period's first day.
func (p Period) From() calendar.Date { return p.from }

// To is the period's last day.
func (p Period) To() calendar.Date { return p.to }

// Start is the instant the period starts in Colombia.
func (p Period) Start() time.Time { return p.from.Start() }

// End is the instant after the period: the start of the day after To.
func (p Period) End() time.Time { return p.to.AddDays(1).Start() }
