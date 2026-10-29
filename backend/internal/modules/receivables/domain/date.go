package domain

import "time"

// bogota is the tenants' time zone: Colombia has no daylight saving time.
var bogota = time.FixedZone("America/Bogota", -5*3600)

// Date is a calendar day, with no time of day or zone.
type Date struct{ t time.Time }

// NewDate returns the day of year, month and day (normalized like
// time.Date).
func NewDate(year int, month time.Month, day int) Date {
	return Date{time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// DateIn returns the calendar day of t in America/Bogota.
func DateIn(t time.Time) Date { return NewDate(t.In(bogota).Date()) }

// DateOf returns the day of t's year, month and day as they read, in any
// zone: how a DATE column comes back from the database.
func DateOf(t time.Time) Date { return NewDate(t.Date()) }

// AddDays returns the day n days after d (before it, for a negative n).
func (d Date) AddDays(n int) Date { return Date{d.t.AddDate(0, 0, n)} }

// DaysSince returns how many days d is after o (negative if before).
func (d Date) DaysSince(o Date) int { return int(d.t.Sub(o.t).Hours() / 24) }

// Time returns the day at midnight UTC, for storage.
func (d Date) Time() time.Time { return d.t }

// IsZero reports whether d is the zero Date.
func (d Date) IsZero() bool { return d.t.IsZero() }

func (d Date) String() string { return d.t.Format(time.DateOnly) }
