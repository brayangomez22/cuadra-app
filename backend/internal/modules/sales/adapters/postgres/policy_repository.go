package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// PolicyRepository implements domain.PolicyRepository on sales_settings.
type PolicyRepository struct{ db *db.DB }

// NewPolicyRepository returns a PolicyRepository on d.
func NewPolicyRepository(d *db.DB) *PolicyRepository { return &PolicyRepository{db: d} }

// Get returns the tenant's policy; the defaults if it has no settings row.
func (r *PolicyRepository) Get(ctx context.Context, tenantID uuid.UUID) (domain.SalesPolicy, error) {
	var policy domain.SalesPolicy
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		maxDiscount, err := sqlcgen.New(tx).GetSalesPolicy(ctx, tenantID)
		if err != nil {
			return err
		}
		policy.MaxDiscount, err = domain.NewDiscountLimit(maxDiscount)
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.SalesPolicy{}, nil
	case err != nil:
		return domain.SalesPolicy{}, fmt.Errorf("sales: get sales policy: %w", err)
	}
	return policy, nil
}
