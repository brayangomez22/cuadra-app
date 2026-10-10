package ratelimit_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/ratelimit"
)

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newLimiter(limit int, window time.Duration) (*ratelimit.FixedWindow, *clock) {
	c := &clock{now: time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)}
	return ratelimit.NewFixedWindow(limit, window, ratelimit.WithClock(c.Now)), c
}

func TestFixedWindow(t *testing.T) {
	t.Run("permite hasta el límite y luego rechaza con el tiempo restante", func(t *testing.T) {
		l, c := newLimiter(3, time.Minute)
		for range 3 {
			ok, _ := l.Allow("203.0.113.7")
			require.True(t, ok)
		}
		c.Advance(20 * time.Second)

		ok, retryAfter := l.Allow("203.0.113.7")

		require.False(t, ok)
		require.Equal(t, 40*time.Second, retryAfter)
	})

	t.Run("las claves son independientes", func(t *testing.T) {
		l, _ := newLimiter(1, time.Minute)
		ok, _ := l.Allow("a")
		require.True(t, ok)
		ok, _ = l.Allow("a")
		require.False(t, ok)

		ok, _ = l.Allow("b")
		require.True(t, ok)
	})

	t.Run("la ventana se reinicia al vencer", func(t *testing.T) {
		l, c := newLimiter(1, time.Minute)
		ok, _ := l.Allow("a")
		require.True(t, ok)
		c.Advance(time.Minute)

		ok, _ = l.Allow("a")
		require.True(t, ok)
	})

	t.Run("olvida las claves vencidas para no crecer sin límite", func(t *testing.T) {
		l, c := newLimiter(5, time.Minute)
		for i := range 100 {
			l.Allow(fmt.Sprintf("key-%d", i))
		}
		require.Equal(t, 100, l.Len())
		c.Advance(time.Minute)

		l.Allow("nueva")

		require.Equal(t, 1, l.Len())
	})

	t.Run("es seguro con llamadas concurrentes", func(t *testing.T) {
		l, _ := newLimiter(50, time.Minute)
		var wg sync.WaitGroup
		var mu sync.Mutex
		allowed := 0
		for range 200 {
			wg.Go(func() {
				if ok, _ := l.Allow("a"); ok {
					mu.Lock()
					allowed++
					mu.Unlock()
				}
			})
		}
		wg.Wait()
		require.Equal(t, 50, allowed)
	})
}
