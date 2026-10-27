package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Transactor implements app.Transactor with db.InTenantTx: every
// repository of any module called with fn's ctx joins the transaction.
type Transactor struct{ db *db.DB }

// NewTransactor returns a Transactor on d.
func NewTransactor(d *db.DB) Transactor { return Transactor{db: d} }

// InTx runs fn in one transaction of tenantID.
func (t Transactor) InTx(ctx context.Context, tenantID uuid.UUID, fn func(context.Context) error) error {
	return t.db.InTenantTx(ctx, tenantID, func(ctx context.Context, _ pgx.Tx) error { return fn(ctx) })
}
