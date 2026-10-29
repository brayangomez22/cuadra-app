package jobs

import (
	"time"

	"github.com/riverqueue/river"
)

// Bogota is the business's time zone for schedules. Colombia has no
// daylight saving time, so a fixed zone needs no tzdata in the image.
var Bogota = time.FixedZone("America/Bogota", -5*3600)

// daily runs once a day at a wall-clock time of a location.
type daily struct {
	hour, minute int
	loc          *time.Location
}

// Daily is the schedule of a periodic job that runs every day at hour:minute
// in loc (Bogota for the business's nights).
func Daily(hour, minute int, loc *time.Location) river.PeriodicSchedule {
	return daily{hour: hour, minute: minute, loc: loc}
}

// Next returns the first hour:minute in loc after current.
func (d daily) Next(current time.Time) time.Time {
	local := current.In(d.loc)
	next := time.Date(local.Year(), local.Month(), local.Day(), d.hour, d.minute, 0, 0, d.loc)
	if !next.After(local) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
