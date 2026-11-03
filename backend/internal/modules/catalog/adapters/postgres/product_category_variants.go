package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
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

// InactiveVariants returns those of variantIDs in tenantID that are no
// longer sold: inactive, or of an archived product.
func (r *ProductRepository) InactiveVariants(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) ([]uuid.UUID, error) {
	ids, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]uuid.UUID, error) {
		return sqlcgen.New(tx).InactiveVariants(ctx, variantIDs)
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: inactive variants: %w", err)
	}
	return ids, nil
}
