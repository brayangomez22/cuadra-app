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

// UnitRepository implements domain.UnitRepository.
type UnitRepository struct{ db *db.DB }

// NewUnitRepository returns a UnitRepository on d.
func NewUnitRepository(d *db.DB) *UnitRepository { return &UnitRepository{db: d} }

// Create stores a new unit; domain.ErrUnitCodeTaken if another unit of the
// tenant has its code.
func (r *UnitRepository) Create(ctx context.Context, u *domain.UnitOfMeasure) error {
	err := r.db.WithTenantTx(ctx, u.TenantID(), func(tx pgx.Tx) error {
		return createUnit(ctx, sqlcgen.New(tx), u)
	})
	switch {
	case violates(err, uniqueViolation, unitsCodeKey):
		return domain.ErrUnitCodeTaken
	case err != nil:
		return fmt.Errorf("catalog: create unit: %w", err)
	}
	return nil
}

func createUnit(ctx context.Context, q *sqlcgen.Queries, u *domain.UnitOfMeasure) error {
	return q.CreateUnit(ctx, sqlcgen.CreateUnitParams{
		ID: u.ID(), TenantID: u.TenantID(), Code: u.Code(), Name: u.Name(), CreatedAt: u.CreatedAt(),
	})
}

// GetByID returns domain.ErrUnitNotFound when the unit does not exist in
// tenantID.
func (r *UnitRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.UnitOfMeasure, error) {
	return r.get(ctx, tenantID, func(q *sqlcgen.Queries) (sqlcgen.UnitsOfMeasure, error) { return q.GetUnit(ctx, id) })
}

// GetByCode returns the unit with exactly code; domain.ErrUnitNotFound if
// tenantID has none.
func (r *UnitRepository) GetByCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.UnitOfMeasure, error) {
	return r.get(ctx, tenantID, func(q *sqlcgen.Queries) (sqlcgen.UnitsOfMeasure, error) { return q.GetUnitByCode(ctx, code) })
}

func (r *UnitRepository) get(ctx context.Context, tenantID uuid.UUID, query func(*sqlcgen.Queries) (sqlcgen.UnitsOfMeasure, error)) (*domain.UnitOfMeasure, error) {
	var row sqlcgen.UnitsOfMeasure
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		row, err = query(sqlcgen.New(tx))
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrUnitNotFound
	case err != nil:
		return nil, fmt.Errorf("catalog: get unit: %w", err)
	}
	return unitFromRow(row)
}

// List returns every unit of the tenant, by name.
func (r *UnitRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.UnitOfMeasure, error) {
	var rows []sqlcgen.UnitsOfMeasure
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		rows, err = sqlcgen.New(tx).ListUnits(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: list units: %w", err)
	}
	out := make([]*domain.UnitOfMeasure, 0, len(rows))
	for _, row := range rows {
		u, err := unitFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

// Update stores the code and the name.
func (r *UnitRepository) Update(ctx context.Context, u *domain.UnitOfMeasure) error {
	var affected int64
	err := r.db.WithTenantTx(ctx, u.TenantID(), func(tx pgx.Tx) error {
		var err error
		affected, err = sqlcgen.New(tx).UpdateUnit(ctx, sqlcgen.UpdateUnitParams{ID: u.ID(), Code: u.Code(), Name: u.Name()})
		return err
	})
	switch {
	case violates(err, uniqueViolation, unitsCodeKey):
		return domain.ErrUnitCodeTaken
	case err != nil:
		return fmt.Errorf("catalog: update unit: %w", err)
	case affected == 0:
		return domain.ErrUnitNotFound
	}
	return nil
}

// Delete removes the unit. The foreign key of the products that use it
// refuses the delete, so a concurrent insert cannot slip in.
func (r *UnitRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	var affected int64
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		affected, err = sqlcgen.New(tx).DeleteUnit(ctx, id)
		return err
	})
	switch {
	case isForeignKeyViolation(err):
		return domain.ErrUnitInUse
	case err != nil:
		return fmt.Errorf("catalog: delete unit: %w", err)
	case affected == 0:
		return domain.ErrUnitNotFound
	}
	return nil
}

func unitFromRow(row sqlcgen.UnitsOfMeasure) (*domain.UnitOfMeasure, error) {
	u, err := domain.RehydrateUnit(domain.UnitSnapshot{
		ID: row.ID, TenantID: row.TenantID, Code: row.Code, Name: row.Name, CreatedAt: row.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: unit %s: %w", row.ID, err)
	}
	return u, nil
}
