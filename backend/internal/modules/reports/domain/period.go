package domain

import (
	"time"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/calendar"
)

// MaxPeriodDays is the longest period of a report: a year, leap or not.
const MaxPeriodDays = 366

// Period is the days a report covers in Colombia, both included. With a
// cut, the last day ends at that time of day instead of at midnight.
type Period struct {
	from, to calendar.Date
	cut      time.Duration
	hasCut   bool
}

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

// End is the instant after the period: the start of the day after To, or
// the cut of To.
func (p Period) End() time.Time {
	if p.hasCut {
		return p.to.Start().Add(p.cut)
	}
	return p.to.AddDays(1).Start()
}

// Until is the period with its last day ending at clock, a time of day in
// Colombia (HH:MM, 00:00 to 23:59): to compare the day so far with another
// day up to the same time. Empty leaves the period whole; anything else
// is ErrInvalidPeriod.
func (p Period) Until(clock string) (Period, error) {
	if clock == "" {
		return p, nil
	}
	at, err := time.Parse("15:04", clock)
	if err != nil || len(clock) != len("15:04") {
		return Period{}, ErrInvalidPeriod
	}
	p.cut = time.Duration(at.Hour())*time.Hour + time.Duration(at.Minute())*time.Minute
	p.hasCut = true
	return p, nil
}
