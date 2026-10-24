// Package postgres implements the catalog repositories on PostgreSQL. Every
// operation runs in db.WithTenantTx, so Row-Level Security limits it to one
// tenant even if a query forgets its filter.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// ProductRepository implements domain.ProductRepository.
type ProductRepository struct{ db *db.DB }

// NewProductRepository returns a ProductRepository on d.
func NewProductRepository(d *db.DB) *ProductRepository { return &ProductRepository{db: d} }

// Create stores a new product and its variants in one transaction.
func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
	attrs, err := attributesJSON(p.Attributes())
	if err != nil {
		return err
	}
	err = r.db.WithTenantTx(ctx, p.TenantID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := q.CreateProduct(ctx, sqlcgen.CreateProductParams{
			ID:          p.ID(),
			TenantID:    p.TenantID(),
			Name:        p.Name(),
			Description: p.Description(),
			CategoryID:  p.CategoryID(),
			BaseUnitID:  p.BaseUnit().ID,
			TaxRate:     p.TaxRate().Rate(),
			Attributes:  attrs,
			Active:      p.IsActive(),
			CreatedAt:   p.CreatedAt(),
		}); err != nil {
			return err
		}
		return saveVariants(ctx, q, p)
	})
	if err := productWriteError(err); err != nil {
		return fmt.Errorf("catalog: create product: %w", err)
	}
	return nil
}

// GetByID returns domain.ErrProductNotFound when the product does not exist
// in tenantID.
func (r *ProductRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	var row sqlcgen.GetProductRow
	var variants []sqlcgen.ListVariantsRow
	var units []sqlcgen.ListVariantUnitsRow
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if row, err = q.GetProduct(ctx, id); err != nil {
			return err
		}
		if variants, err = q.ListVariants(ctx, []uuid.UUID{id}); err != nil {
			return err
		}
		units, err = q.ListVariantUnits(ctx, variantIDs(variants))
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrProductNotFound
	case err != nil:
		return nil, fmt.Errorf("catalog: get product: %w", err)
	}
	return productFromRows(row, variants, unitsByVariant(units))
}

// Update stores every field of the product and of its variants but their
// ids, tenant and creation time, and inserts the variants it does not have
// yet. Variants are never deleted: they are deactivated.
func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
	attrs, err := attributesJSON(p.Attributes())
	if err != nil {
		return err
	}
	err = r.db.WithTenantTx(ctx, p.TenantID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		affected, err := q.UpdateProduct(ctx, sqlcgen.UpdateProductParams{
			ID:          p.ID(),
			Name:        p.Name(),
			Description: p.Description(),
			CategoryID:  p.CategoryID(),
			BaseUnitID:  p.BaseUnit().ID,
			TaxRate:     p.TaxRate().Rate(),
			Attributes:  attrs,
			Active:      p.IsActive(),
		})
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrProductNotFound
		}
		return saveVariants(ctx, q, p)
	})
	if errors.Is(err, domain.ErrProductNotFound) || errors.Is(err, domain.ErrBarcodeTaken) {
		return err
	}
	if err := productWriteError(err); err != nil {
		return fmt.Errorf("catalog: update product: %w", err)
	}
	return nil
}

// VariantProducts returns the product of each of variantIDs that exists in
// tenantID; the others are left out.
func (r *ProductRepository) VariantProducts(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	rows, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]sqlcgen.VariantProductsRow, error) {
		return sqlcgen.New(tx).VariantProducts(ctx, variantIDs)
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: products of variants: %w", err)
	}
	out := make(map[uuid.UUID]uuid.UUID, len(rows))
	for _, row := range rows {
		out[row.ID] = row.ProductID
	}
	return out, nil
}
