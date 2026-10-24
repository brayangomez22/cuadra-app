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

// CategoryRepository implements domain.CategoryRepository.
type CategoryRepository struct{ db *db.DB }

// NewCategoryRepository returns a CategoryRepository on d.
func NewCategoryRepository(d *db.DB) *CategoryRepository { return &CategoryRepository{db: d} }

// Create stores a new category; domain.ErrCategoryNotFound if its parent does
// not exist in the tenant.
func (r *CategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	err := r.db.WithTenantTx(ctx, c.TenantID(), func(tx pgx.Tx) error {
		return createCategory(ctx, sqlcgen.New(tx), c)
	})
	switch {
	case db.Violates(err, db.ForeignKeyViolation, categoriesParentFKey):
		return domain.ErrCategoryNotFound
	case err != nil:
		return fmt.Errorf("catalog: create category: %w", err)
	}
	return nil
}

func createCategory(ctx context.Context, q *sqlcgen.Queries, c *domain.Category) error {
	return q.CreateCategory(ctx, sqlcgen.CreateCategoryParams{
		ID:        c.ID(),
		TenantID:  c.TenantID(),
		Name:      c.Name(),
		ParentID:  c.ParentID(),
		CreatedAt: c.CreatedAt(),
	})
}

// GetByID returns domain.ErrCategoryNotFound when the category does not
// exist in tenantID.
func (r *CategoryRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Category, error) {
	return categoryRows.one(ctx, r.db, tenantID, func(q *sqlcgen.Queries) (sqlcgen.Category, error) { return q.GetCategory(ctx, id) })
}

// List returns every category of the tenant, by name.
func (r *CategoryRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Category, error) {
	return categoryRows.all(ctx, r.db, tenantID, func(q *sqlcgen.Queries) ([]sqlcgen.Category, error) { return q.ListCategories(ctx) })
}

// Update stores the name and parent.
func (r *CategoryRepository) Update(ctx context.Context, c *domain.Category) error {
	// A missing parent violates the foreign key: the category is not found.
	return categoryRows.writeOne(ctx, r.db, c.TenantID(), "update",
		onViolation(db.ForeignKeyViolation, categoriesParentFKey, domain.ErrCategoryNotFound),
		func(q *sqlcgen.Queries) (int64, error) {
			return q.UpdateCategory(ctx, sqlcgen.UpdateCategoryParams{ID: c.ID(), Name: c.Name(), ParentID: c.ParentID()})
		})
}

// Delete removes the category. The foreign keys of its products and
// subcategories refuse the delete, so a concurrent insert cannot slip in.
func (r *CategoryRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return categoryRows.writeOne(ctx, r.db, tenantID, "delete",
		onCode(db.ForeignKeyViolation, domain.ErrCategoryInUse),
		func(q *sqlcgen.Queries) (int64, error) { return q.DeleteCategory(ctx, id) })
}

// Ancestors returns id and its ancestors, nearest first.
func (r *CategoryRepository) Ancestors(ctx context.Context, tenantID, id uuid.UUID) ([]uuid.UUID, error) {
	chain, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]uuid.UUID, error) {
		return sqlcgen.New(tx).CategoryAncestors(ctx, id)
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

var categoryRows = entityRows[sqlcgen.Category, *domain.Category]{
	single: "category", plural: "categories", notFound: domain.ErrCategoryNotFound, fromRow: categoryFromRow,
}
