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

// CategoryRepository implements domain.CategoryRepository.
type CategoryRepository struct{ db *db.DB }

// NewCategoryRepository returns a CategoryRepository on d.
func NewCategoryRepository(d *db.DB) *CategoryRepository { return &CategoryRepository{db: d} }

// Create stores a new category; domain.ErrCategoryNotFound if its parent does
// not exist in the tenant.
func (r *CategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	err := r.db.WithTenantTx(ctx, c.TenantID(), func(tx pgx.Tx) error {
		return sqlcgen.New(tx).CreateCategory(ctx, sqlcgen.CreateCategoryParams{
			ID:        c.ID(),
			TenantID:  c.TenantID(),
			Name:      c.Name(),
			ParentID:  c.ParentID(),
			CreatedAt: c.CreatedAt(),
		})
	})
	switch {
	case violates(err, foreignKeyViolation, categoriesParentFKey):
		return domain.ErrCategoryNotFound
	case err != nil:
		return fmt.Errorf("catalog: create category: %w", err)
	}
	return nil
}

// GetByID returns domain.ErrCategoryNotFound when the category does not
// exist in tenantID.
func (r *CategoryRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Category, error) {
	var row sqlcgen.Category
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		row, err = sqlcgen.New(tx).GetCategory(ctx, id)
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrCategoryNotFound
	case err != nil:
		return nil, fmt.Errorf("catalog: get category: %w", err)
	}
	return categoryFromRow(row)
}

// List returns every category of the tenant, by name.
func (r *CategoryRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Category, error) {
	var rows []sqlcgen.Category
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		rows, err = sqlcgen.New(tx).ListCategories(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: list categories: %w", err)
	}
	out := make([]*domain.Category, 0, len(rows))
	for _, row := range rows {
		c, err := categoryFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// Update stores the name and parent.
func (r *CategoryRepository) Update(ctx context.Context, c *domain.Category) error {
	var affected int64
	err := r.db.WithTenantTx(ctx, c.TenantID(), func(tx pgx.Tx) error {
		var err error
		affected, err = sqlcgen.New(tx).UpdateCategory(ctx, sqlcgen.UpdateCategoryParams{
			ID: c.ID(), Name: c.Name(), ParentID: c.ParentID(),
		})
		return err
	})
	switch {
	case violates(err, foreignKeyViolation, categoriesParentFKey):
		return domain.ErrCategoryNotFound
	case err != nil:
		return fmt.Errorf("catalog: update category: %w", err)
	case affected == 0:
		return domain.ErrCategoryNotFound
	}
	return nil
}

// Delete removes the category. The foreign keys of its products and
// subcategories refuse the delete, so a concurrent insert cannot slip in.
func (r *CategoryRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	var affected int64
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		affected, err = sqlcgen.New(tx).DeleteCategory(ctx, id)
		return err
	})
	switch {
	case isForeignKeyViolation(err):
		return domain.ErrCategoryInUse
	case err != nil:
		return fmt.Errorf("catalog: delete category: %w", err)
	case affected == 0:
		return domain.ErrCategoryNotFound
	}
	return nil
}

// Ancestors returns id and its ancestors, nearest first.
func (r *CategoryRepository) Ancestors(ctx context.Context, tenantID, id uuid.UUID) ([]uuid.UUID, error) {
	var chain []uuid.UUID
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		chain, err = sqlcgen.New(tx).CategoryAncestors(ctx, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: category ancestors: %w", err)
	}
	return chain, nil
}

func categoryFromRow(row sqlcgen.Category) (*domain.Category, error) {
	c, err := domain.RehydrateCategory(domain.CategorySnapshot{
		ID: row.ID, TenantID: row.TenantID, Name: row.Name, ParentID: row.ParentID, CreatedAt: row.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: category %s: %w", row.ID, err)
	}
	return c, nil
}
