package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// PolicyRepository implements domain.PolicyRepository on inventory_settings.
type PolicyRepository struct{ db *db.DB }

// NewPolicyRepository returns a PolicyRepository on d.
func NewPolicyRepository(d *db.DB) *PolicyRepository { return &PolicyRepository{db: d} }

// Get returns the tenant's policy; the defaults if it has no settings row.
func (r *PolicyRepository) Get(ctx context.Context, tenantID uuid.UUID) (domain.StockPolicy, error) {
	allowNegative, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (bool, error) {
		return sqlcgen.New(tx).GetStockPolicy(ctx, tenantID)
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.StockPolicy{}, nil
	case err != nil:
		return domain.StockPolicy{}, fmt.Errorf("inventory: get stock policy: %w", err)
	}
	return domain.StockPolicy{AllowNegative: allowNegative}, nil
}
