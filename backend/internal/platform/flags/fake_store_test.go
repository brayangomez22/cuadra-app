package flags_test

import (
	"context"
	"errors"
	"maps"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags"
)

// fakeStore keeps the flags in memory and counts the reads, to tell when
// the evaluator went to the store and when it used its cache.
type fakeStore struct {
	mu         sync.Mutex
	globals    map[string]bool
	exceptions map[uuid.UUID]map[string]bool
	names      map[uuid.UUID]string
	reads      int
	fail       error
}

var _ flags.Store = (*fakeStore)(nil)

func newFakeStore() *fakeStore {
	return &fakeStore{globals: map[string]bool{}, exceptions: map[uuid.UUID]map[string]bool{}, names: map[uuid.UUID]string{}}
}

func (f *fakeStore) Globals(context.Context) (map[string]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if f.fail != nil {
		return nil, f.fail
	}
	return clone(f.globals), nil
}

func (f *fakeStore) Exceptions(_ context.Context, tenantID uuid.UUID) (map[string]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if f.fail != nil {
		return nil, f.fail
	}
	return clone(f.exceptions[tenantID]), nil
}

func (f *fakeStore) AllExceptions(context.Context) ([]flags.Exception, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []flags.Exception
	for tenant, byKey := range f.exceptions {
		for key, on := range byKey {
			out = append(out, flags.Exception{TenantID: tenant, TenantName: f.names[tenant], Key: key, Enabled: on})
		}
	}
	return out, nil
}

func (f *fakeStore) SetGlobal(_ context.Context, c flags.Change) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.globals[c.Key] = c.Enabled
	return nil
}

func (f *fakeStore) SetException(_ context.Context, tenantID uuid.UUID, c flags.Change) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.names[tenantID]; !ok {
		return flags.ErrUnknownTenant
	}
	if f.exceptions[tenantID] == nil {
		f.exceptions[tenantID] = map[string]bool{}
	}
	f.exceptions[tenantID][c.Key] = c.Enabled
	return nil
}

func (f *fakeStore) ClearException(_ context.Context, tenantID uuid.UUID, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.exceptions[tenantID], key)
	return nil
}

// set changes the store behind the evaluator's back, as another instance
// of the API would.
func (f *fakeStore) setGlobal(key string, on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.globals[key] = on
}

func (f *fakeStore) setException(tenant uuid.UUID, key string, on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.exceptions[tenant] == nil {
		f.exceptions[tenant] = map[string]bool{}
	}
	f.exceptions[tenant][key] = on
}

func (f *fakeStore) addTenant(name string) uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := uuid.Must(uuid.NewV7())
	f.names[id] = name
	return id
}

func (f *fakeStore) readCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads
}

func clone(m map[string]bool) map[string]bool {
	if m == nil {
		return map[string]bool{}
	}
	return maps.Clone(m)
}

var errDown = errors.New("store down")

// clock is a settable time source.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock() *clock { return &clock{now: time.Date(2026, 11, 16, 15, 0, 0, 0, time.UTC)} }

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
