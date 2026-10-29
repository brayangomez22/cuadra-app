package demo

import (
	"sync"
	"time"
)

// Clock is the simulated time of a generation: the API built for the
// generator reads it, so six months of history are written in minutes with
// their own dates.
type Clock struct {
	mu  sync.Mutex
	now time.Time
}

// Now returns the simulated time.
func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Set moves the clock to t; it never goes back, so the times the API
// stores keep their order.
func (c *Clock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t.After(c.now) {
		c.now = t
	}
}
