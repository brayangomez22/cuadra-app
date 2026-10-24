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
		if err := createAll(ctx, q, seed.Units, createUnit); err != nil {
			return err
		}
		if err := createAll(ctx, q, seed.Categories, createCategory); err != nil {
			return err
		}
		if err := createAll(ctx, q, seed.Attributes, createAttribute); err != nil {
			return err
		}
		return createAll(ctx, q, seed.VariantAxes, createVariantAxis)
	})
	return seedError(err)
}

// createAll stores items in order, with the create of their repository.
func createAll[T any](ctx context.Context, q *sqlcgen.Queries, items []T, create func(context.Context, *sqlcgen.Queries, T) error) error {
	for _, item := range items {
		if err := create(ctx, q, item); err != nil {
			return err
		}
	}
	return nil
}

// seedError translates the errors of a seed as each repository's Create does.
func seedError(err error) error {
	switch {
	case db.Violates(err, db.UniqueViolation, unitsCodeKey):
		return domain.ErrUnitCodeTaken
	case db.Violates(err, db.ForeignKeyViolation, categoriesParentFKey):
		return domain.ErrCategoryNotFound
	case db.Violates(err, db.UniqueViolation, attributesKeyKey):
		return domain.ErrAttributeKeyTaken
	case db.Violates(err, db.UniqueViolation, axesNameKey):
		return domain.ErrAxisNameTaken
	case err != nil:
		return fmt.Errorf("catalog: seed tenant: %w", err)
	}
	return nil
}
