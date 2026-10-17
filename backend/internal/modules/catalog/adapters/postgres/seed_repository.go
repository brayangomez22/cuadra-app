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

// SeedRepository implements domain.SeedRepository.
type SeedRepository struct{ db *db.DB }

// NewSeedRepository returns a SeedRepository on d.
func NewSeedRepository(d *db.DB) *SeedRepository { return &SeedRepository{db: d} }

// Seed stores the seed of tenantID in one transaction (or in the caller's,
// with a ctx from db.InTenantTx), with the errors of each repository's
// Create: either all of it is stored or nothing is. Categories go before
// anything that could reference them.
func (r *SeedRepository) Seed(ctx context.Context, tenantID uuid.UUID, seed domain.TenantSeed) error {
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		for _, u := range seed.Units {
			if err := createUnit(ctx, q, u); err != nil {
				return err
			}
		}
		for _, c := range seed.Categories {
			if err := createCategory(ctx, q, c); err != nil {
				return err
			}
		}
		for _, d := range seed.Attributes {
			if err := createAttribute(ctx, q, d); err != nil {
				return err
			}
		}
		for _, a := range seed.VariantAxes {
			if err := createVariantAxis(ctx, q, a); err != nil {
				return err
			}
		}
		return nil
	})
	switch {
	case violates(err, uniqueViolation, unitsCodeKey):
		return domain.ErrUnitCodeTaken
	case violates(err, foreignKeyViolation, categoriesParentFKey):
		return domain.ErrCategoryNotFound
	case violates(err, uniqueViolation, attributesKeyKey):
		return domain.ErrAttributeKeyTaken
	case violates(err, uniqueViolation, axesNameKey):
		return domain.ErrAxisNameTaken
	case err != nil:
		return fmt.Errorf("catalog: seed tenant: %w", err)
	}
	return nil
}
