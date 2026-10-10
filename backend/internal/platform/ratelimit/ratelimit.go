// Package ratelimit counts attempts per key in memory.
//
// The state lives in one process: with several replicas each one counts on
// its own, so the effective limit is multiplied by the replica count. Callers
// depend on a small interface (Allow), so a shared store (Redis) can replace
// it when a measurement justifies it (see docs/decisiones.md).
package ratelimit

import (
	"sync"
	"time"
)

// FixedWindow allows up to limit attempts per key in each window, starting
// at the key's first attempt. It is safe for concurrent use.
type FixedWindow struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu        sync.Mutex
	buckets   map[string]bucket
	nextSweep time.Time
}

type bucket struct {
	count   int
	resetAt time.Time
}

// Option configures a FixedWindow.
type Option func(*FixedWindow)

// WithClock replaces time.Now (tests).
func WithClock(now func() time.Time) Option { return func(l *FixedWindow) { l.now = now } }

// NewFixedWindow returns a limiter of limit attempts per window.
func NewFixedWindow(limit int, window time.Duration, opts ...Option) *FixedWindow {
	l := &FixedWindow{limit: limit, window: window, now: time.Now, buckets: map[string]bucket{}}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Allow counts an attempt for key. When the limit is reached it returns false
// and how long until the key's window resets.
func (l *FixedWindow) Allow(key string) (allowed bool, retryAfter time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok || !now.Before(b.resetAt) {
		b = bucket{resetAt: now.Add(l.window)}
	}
	if b.count >= l.limit {
		return false, b.resetAt.Sub(now)
	}
	b.count++
	l.buckets[key] = b
	return true, 0
}

// Len returns how many keys are tracked.
func (l *FixedWindow) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}

// sweep drops expired keys at most once per window, so memory is bounded by
// the keys seen in about two windows.
func (l *FixedWindow) sweep(now time.Time) {
	if now.Before(l.nextSweep) {
		return
	}
	for key, b := range l.buckets {
		if !now.Before(b.resetAt) {
			delete(l.buckets, key)
		}
	}
	l.nextSweep = now.Add(l.window)
}
