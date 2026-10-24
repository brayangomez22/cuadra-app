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
	case db.Violates(err, db.UniqueViolation, unitsCodeKey):
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

func (r *UnitRepository) get(ctx context.Context, tenantID uuid.UUID, query func(*sqlcgen.Queries) (sqlcgen.UnitsOfMeasure, error)) (*domain.UnitOfMeasure, error) {
	return unitRows.one(ctx, r.db, tenantID, query)
}

// List returns every unit of the tenant, by name.
func (r *UnitRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.UnitOfMeasure, error) {
	return unitRows.all(ctx, r.db, tenantID, func(q *sqlcgen.Queries) ([]sqlcgen.UnitsOfMeasure, error) { return q.ListUnits(ctx) })
}

// Update stores the code and the name.
func (r *UnitRepository) Update(ctx context.Context, u *domain.UnitOfMeasure) error {
	return unitRows.writeOne(ctx, r.db, u.TenantID(), "update",
		onViolation(db.UniqueViolation, unitsCodeKey, domain.ErrUnitCodeTaken),
		func(q *sqlcgen.Queries) (int64, error) {
			return q.UpdateUnit(ctx, sqlcgen.UpdateUnitParams{ID: u.ID(), Code: u.Code(), Name: u.Name()})
		})
}

// Delete removes the unit. The foreign key of the products that use it
// refuses the delete, so a concurrent insert cannot slip in.
func (r *UnitRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return unitRows.writeOne(ctx, r.db, tenantID, "delete",
		onCode(db.ForeignKeyViolation, domain.ErrUnitInUse),
		func(q *sqlcgen.Queries) (int64, error) { return q.DeleteUnit(ctx, id) })
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

var unitRows = entityRows[sqlcgen.UnitsOfMeasure, *domain.UnitOfMeasure]{
	single: "unit", plural: "units", notFound: domain.ErrUnitNotFound, fromRow: unitFromRow,
}
