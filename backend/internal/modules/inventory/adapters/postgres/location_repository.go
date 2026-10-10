// Package postgres implements the inventory repositories on PostgreSQL. Every
// operation runs in db.WithTenantTx, so Row-Level Security limits it to one
// tenant even if a query forgets its filter.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// LocationRepository implements domain.LocationRepository.
type LocationRepository struct{ db *db.DB }

// NewLocationRepository returns a LocationRepository on d.
func NewLocationRepository(d *db.DB) *LocationRepository { return &LocationRepository{db: d} }

// Create stores a new location.
func (r *LocationRepository) Create(ctx context.Context, l *domain.Location) error {
	err := r.db.WithTenantTx(ctx, l.TenantID(), func(tx pgx.Tx) error {
		return sqlcgen.New(tx).CreateLocation(ctx, sqlcgen.CreateLocationParams{
			ID:        l.ID(),
			TenantID:  l.TenantID(),
			Name:      l.Name(),
			Active:    l.IsActive(),
			CreatedAt: l.CreatedAt(),
		})
	})
	switch {
	case violates(err, uniqueViolation, locationsNameKey):
		return domain.ErrLocationNameTaken
	case err != nil:
		return fmt.Errorf("inventory: create location: %w", err)
	}
	return nil
}

// GetByID returns domain.ErrLocationNotFound when the location does not
// exist in tenantID.
func (r *LocationRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Location, error) {
	var row sqlcgen.Location
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		row, err = sqlcgen.New(tx).GetLocation(ctx, id)
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrLocationNotFound
	case err != nil:
		return nil, fmt.Errorf("inventory: get location: %w", err)
	}
	return locationFromRow(row)
}

// List returns every location of the tenant, by name.
func (r *LocationRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Location, error) {
	var rows []sqlcgen.Location
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		rows, err = sqlcgen.New(tx).ListLocations(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("inventory: list locations: %w", err)
	}
	out := make([]*domain.Location, 0, len(rows))
	for _, row := range rows {
		l, err := locationFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

func locationFromRow(row sqlcgen.Location) (*domain.Location, error) {
	l, err := domain.RehydrateLocation(domain.LocationSnapshot{
		ID: row.ID, TenantID: row.TenantID, Name: row.Name, Active: row.Active, CreatedAt: row.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("inventory: location %s: %w", row.ID, err)
	}
	return l, nil
}
