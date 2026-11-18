package app_test

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/app"
)

// errAudit is the failure of a fakeAudit made to fail.
var errAudit = errors.New("audit down")

// fakeAudit records the audit entries in memory.
type fakeAudit struct {
	entries []app.AuditEntry
	fail    bool
}

func (f *fakeAudit) Record(_ context.Context, _, _ uuid.UUID, e app.AuditEntry) error {
	if f.fail {
		return errAudit
	}
	f.entries = append(f.entries, e)
	return nil
}

// fakeTx runs fn without a transaction: the fakes have none.
type fakeTx struct{}

func (fakeTx) InTx(ctx context.Context, _ uuid.UUID, fn func(context.Context) error) error {
	return fn(ctx)
}
