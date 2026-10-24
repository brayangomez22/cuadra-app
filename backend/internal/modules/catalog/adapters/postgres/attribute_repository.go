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

// AttributeRepository implements domain.AttributeRepository.
type AttributeRepository struct{ db *db.DB }

// NewAttributeRepository returns an AttributeRepository on d.
func NewAttributeRepository(d *db.DB) *AttributeRepository { return &AttributeRepository{db: d} }

// Create stores a new definition; domain.ErrAttributeKeyTaken if another
// definition of the tenant has its key.
func (r *AttributeRepository) Create(ctx context.Context, d *domain.AttributeDefinition) error {
	err := r.db.WithTenantTx(ctx, d.TenantID(), func(tx pgx.Tx) error {
		return createAttribute(ctx, sqlcgen.New(tx), d)
	})
	switch {
	case db.Violates(err, db.UniqueViolation, attributesKeyKey):
		return domain.ErrAttributeKeyTaken
	case err != nil:
		return fmt.Errorf("catalog: create attribute: %w", err)
	}
	return nil
}

func createAttribute(ctx context.Context, q *sqlcgen.Queries, d *domain.AttributeDefinition) error {
	return q.CreateAttribute(ctx, sqlcgen.CreateAttributeParams{
		ID: d.ID(), TenantID: d.TenantID(), Key: d.Key(), Label: d.Label(), Type: string(d.Type()),
		Options: nonNil(d.Options()), Required: d.Required(), CreatedAt: d.CreatedAt(),
	})
}

// GetByID returns domain.ErrAttributeNotFound when the definition does not
// exist in tenantID.
func (r *AttributeRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.AttributeDefinition, error) {
	return attributeRows.one(ctx, r.db, tenantID, func(q *sqlcgen.Queries) (sqlcgen.AttributeDefinition, error) { return q.GetAttribute(ctx, id) })
}

// List returns every definition of the tenant, by label.
func (r *AttributeRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.AttributeDefinition, error) {
	return attributeRows.all(ctx, r.db, tenantID, func(q *sqlcgen.Queries) ([]sqlcgen.AttributeDefinition, error) { return q.ListAttributes(ctx) })
}

// Update stores the label, the options and whether the attribute is
// required; domain.ErrAttributeOptionInUse if a product has one of
// removedOptions. The check runs after the update, so the definition's row
// stays locked until the transaction ends.
func (r *AttributeRepository) Update(ctx context.Context, d *domain.AttributeDefinition, removedOptions []string) error {
	return attributeRows.writeTx(ctx, r.db, d.TenantID(), "update", func(q *sqlcgen.Queries) error {
		affected, err := q.UpdateAttribute(ctx, sqlcgen.UpdateAttributeParams{
			ID: d.ID(), Label: d.Label(), Options: nonNil(d.Options()), Required: d.Required(),
		})
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrAttributeNotFound
		}
		if len(removedOptions) == 0 {
			return nil
		}
		used, err := q.AttributeOptionsInUse(ctx, sqlcgen.AttributeOptionsInUseParams{Key: d.Key(), Options: removedOptions})
		return failIfUsed(used, err, domain.ErrAttributeOptionInUse)
	}, domain.ErrAttributeOptionInUse)
}

// Delete removes the definition; domain.ErrAttributeInUse if a product has a
// value for it. Products keep their values by key, with no foreign key, so
// the check runs after the delete, in its transaction.
func (r *AttributeRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return attributeRows.writeTx(ctx, r.db, tenantID, "delete", func(q *sqlcgen.Queries) error {
		key, err := q.DeleteAttribute(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAttributeNotFound
		}
		if err != nil {
			return err
		}
		used, err := q.AttributeInUse(ctx, key)
		return failIfUsed(used, err, domain.ErrAttributeInUse)
	}, domain.ErrAttributeInUse)
}

func attributeFromRow(row sqlcgen.AttributeDefinition) (*domain.AttributeDefinition, error) {
	d, err := domain.RehydrateAttributeDefinition(domain.AttributeSnapshot{
		ID: row.ID, TenantID: row.TenantID, Key: row.Key, Label: row.Label, Type: domain.AttributeType(row.Type),
		Options: row.Options, Required: row.Required, CreatedAt: row.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: attribute %s: %w", row.ID, err)
	}
	return d, nil
}

// nonNil stores a missing list as an empty array: the column is NOT NULL.
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

var attributeRows = entityRows[sqlcgen.AttributeDefinition, *domain.AttributeDefinition]{
	single: "attribute", plural: "attributes", notFound: domain.ErrAttributeNotFound, fromRow: attributeFromRow,
}
