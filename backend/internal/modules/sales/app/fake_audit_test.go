package app_test

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
)

// fakeAudit records the audit entries in memory.
type fakeAudit struct {
	mu      sync.Mutex
	entries []app.AuditEntry
	// fail makes Record fail with it.
	fail error
}

func (f *fakeAudit) Record(_ context.Context, _, _ uuid.UUID, e app.AuditEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail != nil {
		return f.fail
	}
	f.entries = append(f.entries, e)
	return nil
}

// of returns the recorded entries of action.
func (f *fakeAudit) of(action string) []app.AuditEntry {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []app.AuditEntry
	for _, e := range f.entries {
		if e.Action == action {
			out = append(out, e)
		}
	}
	return out
}

// fakeTx runs fn without a transaction: the fakes have none.
type fakeTx struct{}

func (fakeTx) InTx(ctx context.Context, _ uuid.UUID, fn func(context.Context) error) error {
	return fn(ctx)
}
