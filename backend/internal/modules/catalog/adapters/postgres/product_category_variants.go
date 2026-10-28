package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// VariantsInCategories returns the variants, active or not, of the products
// in categoryIDs or in their subcategories; domain.ErrCategoryNotFound if a
// category does not exist in tenantID. Both reads share a transaction.
func (r *ProductRepository) VariantsInCategories(ctx context.Context, tenantID uuid.UUID, categoryIDs []uuid.UUID) ([]uuid.UUID, error) {
	distinct := make(map[uuid.UUID]struct{}, len(categoryIDs))
	for _, id := range categoryIDs {
		distinct[id] = struct{}{}
	}
	var found int64
	var variants []uuid.UUID
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if found, err = q.CountCategories(ctx, categoryIDs); err != nil {
			return err
		}
		variants, err = q.VariantsInCategories(ctx, categoryIDs)
		return err
	})
	switch {
	case err != nil:
		return nil, fmt.Errorf("catalog: variants in categories: %w", err)
	case int(found) != len(distinct):
		return nil, domain.ErrCategoryNotFound
	}
	return variants, nil
}
