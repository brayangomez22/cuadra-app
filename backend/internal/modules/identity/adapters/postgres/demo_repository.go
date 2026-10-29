package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// The SECURITY DEFINER functions of migration 00027, written by hand like
// loginCandidatesQuery: they run as demo_purge, outside any tenant.
const (
	demoTenantsQuery = `SELECT demo_tenants($1)`
	purgeDemoQuery   = `SELECT purge_demo_tenant($1)`
)

// DemoTenantRepository implements domain.DemoTenantRepository.
type DemoTenantRepository struct{ db *db.DB }

// NewDemoTenantRepository returns a DemoTenantRepository on d.
func NewDemoTenantRepository(d *db.DB) *DemoTenantRepository { return &DemoTenantRepository{db: d} }

// FindByTemplate returns the ids of the demo tenants of template, ordered.
func (r *DemoTenantRepository) FindByTemplate(ctx context.Context, template string) ([]uuid.UUID, error) {
	var out []uuid.UUID
	// Not a tenant transaction: the tenants are what it looks for.
	err := r.db.WithTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, demoTenantsQuery, template)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("identity: find demo tenants: %w", err)
	}
	return out, nil
}

// Purge deletes the demo tenant and all its rows in one transaction;
// domain.ErrNotDemoTenant or domain.ErrTenantNotFound delete nothing.
func (r *DemoTenantRepository) Purge(ctx context.Context, id uuid.UUID) error {
	var result string
	err := r.db.WithTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, purgeDemoQuery, id).Scan(&result)
	})
	if err != nil {
		return fmt.Errorf("identity: purge demo tenant: %w", err)
	}
	switch result {
	case "purged":
		return nil
	case "not_demo":
		return domain.ErrNotDemoTenant
	case "not_found":
		return domain.ErrTenantNotFound
	}
	return fmt.Errorf("identity: purge demo tenant: unexpected result %q", result)
}
