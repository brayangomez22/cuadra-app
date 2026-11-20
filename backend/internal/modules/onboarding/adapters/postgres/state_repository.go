package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// StateRepository implements domain.StateReader on the tables of the other
// modules (tenants, products, stock, users, sales...), read-only.
type StateRepository struct {
	db  *db.DB
	now func() time.Time
}

var _ domain.StateReader = (*StateRepository)(nil)

// NewStateRepository builds the reader on d; now decides which invitations
// are still in force.
func NewStateRepository(d *db.DB, now func() time.Time) *StateRepository {
	return &StateRepository{db: d, now: now}
}

// TenantState reads the tenant's template and the facts of its data in one
// query.
func (r *StateRepository) TenantState(ctx context.Context, tenantID uuid.UUID) (domain.TenantState, error) {
	row, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (sqlcgen.TenantStateRow, error) {
		return sqlcgen.New(tx).TenantState(ctx, sqlcgen.TenantStateParams{TenantID: tenantID, Now: r.now()})
	})
	if err != nil {
		return domain.TenantState{}, fmt.Errorf("onboarding: read tenant state: %w", err)
	}
	return domain.TenantState{Template: row.BusinessTemplate, Facts: domain.Facts{
		HasContact:        row.HasContact,
		HasProduct:        row.HasProduct,
		HasStock:          row.HasStock,
		HasAlternateUnit:  row.HasAlternateUnit,
		HasVariantProduct: row.HasVariantProduct,
		HasReorderRule:    row.HasReorderRule,
		HasCreditCustomer: row.HasCreditCustomer,
		HasTeammate:       row.HasTeammate,
		HasSale:           row.HasSale,
	}}, nil
}
