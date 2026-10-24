package domain

import (
	"time"

	"github.com/google/uuid"
)

// What the forbidigo rules reject in a domain: a float for an amount, the
// clock read directly and a UUID v4.
var price float64 = 1

func stamp() (time.Time, uuid.UUID) { return time.Now(), uuid.New() }

var _ = price
var _ = stamp
