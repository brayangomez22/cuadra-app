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
		return sqlcgen.New(tx).CreateAttribute(ctx, sqlcgen.CreateAttributeParams{
			ID: d.ID(), TenantID: d.TenantID(), Key: d.Key(), Label: d.Label(), Type: string(d.Type()),
			Options: nonNil(d.Options()), Required: d.Required(), CreatedAt: d.CreatedAt(),
		})
	})
	switch {
	case violates(err, uniqueViolation, attributesKeyKey):
		return domain.ErrAttributeKeyTaken
	case err != nil:
		return fmt.Errorf("catalog: create attribute: %w", err)
	}
	return nil
}

// GetByID returns domain.ErrAttributeNotFound when the definition does not
// exist in tenantID.
func (r *AttributeRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.AttributeDefinition, error) {
	var row sqlcgen.AttributeDefinition
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		row, err = sqlcgen.New(tx).GetAttribute(ctx, id)
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrAttributeNotFound
	case err != nil:
		return nil, fmt.Errorf("catalog: get attribute: %w", err)
	}
	return attributeFromRow(row)
}

// List returns every definition of the tenant, by label.
func (r *AttributeRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.AttributeDefinition, error) {
	var rows []sqlcgen.AttributeDefinition
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		rows, err = sqlcgen.New(tx).ListAttributes(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: list attributes: %w", err)
	}
	out := make([]*domain.AttributeDefinition, 0, len(rows))
	for _, row := range rows {
		d, err := attributeFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

// Update stores the label, the options and whether the attribute is
// required; domain.ErrAttributeOptionInUse if a product has one of
// removedOptions. The check runs after the update, so the definition's row
// stays locked until the transaction ends.
func (r *AttributeRepository) Update(ctx context.Context, d *domain.AttributeDefinition, removedOptions []string) error {
	err := r.db.WithTenantTx(ctx, d.TenantID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
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
		if err != nil {
			return err
		}
		if used {
			return domain.ErrAttributeOptionInUse
		}
		return nil
	})
	switch {
	case errors.Is(err, domain.ErrAttributeNotFound), errors.Is(err, domain.ErrAttributeOptionInUse):
		return err
	case err != nil:
		return fmt.Errorf("catalog: update attribute: %w", err)
	}
	return nil
}

// Delete removes the definition; domain.ErrAttributeInUse if a product has a
// value for it. Products keep their values by key, with no foreign key, so
// the check runs after the delete, in its transaction.
func (r *AttributeRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		key, err := q.DeleteAttribute(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAttributeNotFound
		}
		if err != nil {
			return err
		}
		used, err := q.AttributeInUse(ctx, key)
		if err != nil {
			return err
		}
		if used {
			return domain.ErrAttributeInUse
		}
		return nil
	})
	switch {
	case errors.Is(err, domain.ErrAttributeNotFound), errors.Is(err, domain.ErrAttributeInUse):
		return err
	case err != nil:
		return fmt.Errorf("catalog: delete attribute: %w", err)
	}
	return nil
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
