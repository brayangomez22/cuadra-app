package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Transactor implements the Transactor port of a module's app with
// InTenantTx: every repository of any module called with fn's ctx joins
// the transaction. The imports use it for each batch, and the audited use
// cases to store the change and its audit event together.
type Transactor struct{ db *DB }

// NewTransactor returns a Transactor on d.
func NewTransactor(d *DB) Transactor { return Transactor{db: d} }

// InTx runs fn in one transaction of tenantID.
func (t Transactor) InTx(ctx context.Context, tenantID uuid.UUID, fn func(context.Context) error) error {
	return t.db.InTenantTx(ctx, tenantID, func(ctx context.Context, _ pgx.Tx) error { return fn(ctx) })
}
