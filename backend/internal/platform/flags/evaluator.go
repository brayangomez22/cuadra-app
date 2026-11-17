package flags

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// EvaluatorDeps are the Evaluator's dependencies. Now defaults to time.Now
// and Logger to a discarding logger.
type EvaluatorDeps struct {
	Registry *Registry
	Store    Store
	// TTL is how long a read of the store is reused: a change made by
	// another instance of the API shows up here after at most TTL.
	TTL    time.Duration
	Now    func() time.Time
	Logger *slog.Logger
}

// Evaluator decides whether a flag is on for a tenant: the tenant's
// exception wins over the global value, and a flag with neither, or one the
// code does not declare, is off. It keeps what it reads in memory for TTL,
// the global values once and the exceptions per tenant.
type Evaluator struct {
	EvaluatorDeps
	mu      sync.Mutex
	globals cached
	tenants map[uuid.UUID]cached
}

// cached is a read of the store and when it stops being valid.
type cached struct {
	values  map[string]bool
	expires time.Time
}

// NewEvaluator builds an Evaluator.
func NewEvaluator(d EvaluatorDeps) (*Evaluator, error) {
	if usecase.AnyNil(d.Registry, d.Store) {
		return nil, errors.New("flags: missing evaluator dependency")
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	d.Logger = usecase.Logger(d.Logger)
	return &Evaluator{EvaluatorDeps: d, tenants: map[uuid.UUID]cached{}}, nil
}

// Enabled reports whether the flag key is on for the tenant. When the store
// cannot be read the flag is off, the safe value, and the failure is logged:
// a flag never breaks the operation that asks for it.
func (e *Evaluator) Enabled(ctx context.Context, tenantID uuid.UUID, key string) bool {
	on, err := e.enabled(ctx, tenantID, key)
	if err != nil {
		e.Logger.WarnContext(ctx, "feature flag read failed", slog.String("flag", key), slog.Any("error", err))
		return false
	}
	return on
}

// Active lists the declared flags that are on for the tenant, in
// declaration order. It runs on every GET /me, so it opens no span: the
// queries' spans are enough, and most calls hit the cache.
func (e *Evaluator) Active(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	globals, exceptions, err := e.read(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, d := range e.Registry.defs {
		if resolve(globals, exceptions, d.Key) {
			out = append(out, d.Key)
		}
	}
	return out, nil
}

// Invalidate drops the cache, so this instance sees its own changes at once.
func (e *Evaluator) Invalidate() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.globals = cached{}
	clear(e.tenants)
}

func (e *Evaluator) enabled(ctx context.Context, tenantID uuid.UUID, key string) (bool, error) {
	if !e.Registry.Known(key) {
		return false, nil
	}
	globals, exceptions, err := e.read(ctx, tenantID)
	if err != nil {
		return false, err
	}
	return resolve(globals, exceptions, key), nil
}

func resolve(globals, exceptions map[string]bool, key string) bool {
	if on, ok := exceptions[key]; ok {
		return on
	}
	return globals[key]
}

// read returns the global values and the tenant's exceptions, from the
// cache while they are fresh. The store is read outside the lock: two
// callers may both read it when the cache expires, which is harmless.
func (e *Evaluator) read(ctx context.Context, tenantID uuid.UUID) (globals, exceptions map[string]bool, err error) {
	now := e.Now()
	e.mu.Lock()
	g, gOK := e.globals, now.Before(e.globals.expires)
	t, tOK := e.tenants[tenantID]
	tOK = tOK && now.Before(t.expires)
	e.mu.Unlock()

	if !gOK {
		if g.values, err = e.Store.Globals(ctx); err != nil {
			return nil, nil, err
		}
		g.expires = now.Add(e.TTL)
	}
	if !tOK {
		if t.values, err = e.Store.Exceptions(ctx, tenantID); err != nil {
			return nil, nil, err
		}
		t.expires = now.Add(e.TTL)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.globals = g
	e.prune(now)
	e.tenants[tenantID] = t
	return g.values, t.values, nil
}

// prune drops the expired tenants, so the cache holds only the tenants seen
// in the last TTL.
func (e *Evaluator) prune(now time.Time) {
	for id, c := range e.tenants {
		if !now.Before(c.expires) {
			delete(e.tenants, id)
		}
	}
}
