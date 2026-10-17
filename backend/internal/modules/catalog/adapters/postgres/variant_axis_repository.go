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

// VariantAxisRepository implements domain.VariantAxisRepository.
type VariantAxisRepository struct{ db *db.DB }

// NewVariantAxisRepository returns a VariantAxisRepository on d.
func NewVariantAxisRepository(d *db.DB) *VariantAxisRepository { return &VariantAxisRepository{db: d} }

// Create stores a new axis; domain.ErrAxisNameTaken if another axis of the
// tenant has its name.
func (r *VariantAxisRepository) Create(ctx context.Context, a *domain.VariantAxis) error {
	err := r.db.WithTenantTx(ctx, a.TenantID(), func(tx pgx.Tx) error {
		return createVariantAxis(ctx, sqlcgen.New(tx), a)
	})
	switch {
	case violates(err, uniqueViolation, axesNameKey):
		return domain.ErrAxisNameTaken
	case err != nil:
		return fmt.Errorf("catalog: create variant axis: %w", err)
	}
	return nil
}

func createVariantAxis(ctx context.Context, q *sqlcgen.Queries, a *domain.VariantAxis) error {
	return q.CreateVariantAxis(ctx, sqlcgen.CreateVariantAxisParams{
		ID: a.ID(), TenantID: a.TenantID(), Name: a.Name(), AllowedValues: a.Values(), CreatedAt: a.CreatedAt(),
	})
}

// GetByID returns domain.ErrAxisNotFound when the axis does not exist in
// tenantID.
func (r *VariantAxisRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.VariantAxis, error) {
	var row sqlcgen.VariantAxis
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		row, err = sqlcgen.New(tx).GetVariantAxis(ctx, id)
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrAxisNotFound
	case err != nil:
		return nil, fmt.Errorf("catalog: get variant axis: %w", err)
	}
	return axisFromRow(row)
}

// List returns every axis of the tenant, by name.
func (r *VariantAxisRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.VariantAxis, error) {
	var rows []sqlcgen.VariantAxis
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		rows, err = sqlcgen.New(tx).ListVariantAxes(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: list variant axes: %w", err)
	}
	out := make([]*domain.VariantAxis, 0, len(rows))
	for _, row := range rows {
		a, err := axisFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// Update stores the allowed values; domain.ErrAxisValueInUse if a variant
// has one of removedValues, in any case. The check runs after the update,
// so the axis's row stays locked until the transaction ends.
func (r *VariantAxisRepository) Update(ctx context.Context, a *domain.VariantAxis, removedValues []string) error {
	err := r.db.WithTenantTx(ctx, a.TenantID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		affected, err := q.UpdateVariantAxis(ctx, sqlcgen.UpdateVariantAxisParams{ID: a.ID(), AllowedValues: a.Values()})
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrAxisNotFound
		}
		if len(removedValues) == 0 {
			return nil
		}
		used, err := q.VariantAxisValuesInUse(ctx, sqlcgen.VariantAxisValuesInUseParams{Name: a.Name(), AxisValues: removedValues})
		if err != nil {
			return err
		}
		if used {
			return domain.ErrAxisValueInUse
		}
		return nil
	})
	switch {
	case errors.Is(err, domain.ErrAxisNotFound), errors.Is(err, domain.ErrAxisValueInUse):
		return err
	case err != nil:
		return fmt.Errorf("catalog: update variant axis: %w", err)
	}
	return nil
}

// Delete removes the axis; domain.ErrAxisInUse if a variant has a value on
// it. Variants keep their values by name, with no foreign key, so the check
// runs after the delete, in its transaction.
func (r *VariantAxisRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		name, err := q.DeleteVariantAxis(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAxisNotFound
		}
		if err != nil {
			return err
		}
		used, err := q.VariantAxisInUse(ctx, name)
		if err != nil {
			return err
		}
		if used {
			return domain.ErrAxisInUse
		}
		return nil
	})
	switch {
	case errors.Is(err, domain.ErrAxisNotFound), errors.Is(err, domain.ErrAxisInUse):
		return err
	case err != nil:
		return fmt.Errorf("catalog: delete variant axis: %w", err)
	}
	return nil
}

func axisFromRow(row sqlcgen.VariantAxis) (*domain.VariantAxis, error) {
	a, err := domain.RehydrateVariantAxis(domain.VariantAxisSnapshot{
		ID: row.ID, TenantID: row.TenantID, Name: row.Name, Values: row.AllowedValues, CreatedAt: row.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: variant axis %s: %w", row.ID, err)
	}
	return a, nil
}
