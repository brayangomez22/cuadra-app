// Package postgres implements the identity repositories on PostgreSQL. Every
// operation runs in db.WithTenantTx, so Row-Level Security limits it to one
// tenant even if a query forgets its filter.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// TenantRepository implements domain.TenantRepository. A tenant is scoped to
// itself: its id is the transaction's tenant.
type TenantRepository struct{ db *db.DB }

// NewTenantRepository returns a TenantRepository on d.
func NewTenantRepository(d *db.DB) *TenantRepository { return &TenantRepository{db: d} }

// Create stores a new tenant.
func (r *TenantRepository) Create(ctx context.Context, t *domain.Tenant) error {
	err := r.db.WithTenantTx(ctx, t.ID(), func(tx pgx.Tx) error {
		return createTenant(ctx, sqlcgen.New(tx), t)
	})
	if err != nil {
		return fmt.Errorf("identity: create tenant: %w", err)
	}
	return nil
}

// CreateWithOwner stores a new tenant and its first user in one transaction
// scoped to the new tenant: either both are stored or neither is.
func (r *TenantRepository) CreateWithOwner(ctx context.Context, t *domain.Tenant, owner *domain.User) error {
	err := r.db.WithTenantTx(ctx, t.ID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := createTenant(ctx, q, t); err != nil {
			return err
		}
		return createUser(ctx, q, owner)
	})
	switch {
	case isUniqueViolation(err, usersEmailKey):
		return domain.ErrEmailTaken
	case err != nil:
		return fmt.Errorf("identity: create tenant with owner: %w", err)
	}
	return nil
}

func createTenant(ctx context.Context, q *sqlcgen.Queries, t *domain.Tenant) error {
	return q.CreateTenant(ctx, sqlcgen.CreateTenantParams{
		ID:        t.ID(),
		Name:      t.Name(),
		Nit:       t.NIT().String(),
		Status:    string(t.Status()),
		CreatedAt: t.CreatedAt(),
	})
}

// GetByID returns domain.ErrTenantNotFound when the tenant does not exist.
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	var row sqlcgen.Tenant
	err := r.db.WithTenantTx(ctx, id, func(tx pgx.Tx) error {
		var err error
		row, err = sqlcgen.New(tx).GetTenant(ctx, id)
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrTenantNotFound
	case err != nil:
		return nil, fmt.Errorf("identity: get tenant: %w", err)
	}
	return tenantFromRow(row)
}

// Update stores the name and status; domain.ErrTenantNotFound if the tenant
// does not exist.
func (r *TenantRepository) Update(ctx context.Context, t *domain.Tenant) error {
	var affected int64
	err := r.db.WithTenantTx(ctx, t.ID(), func(tx pgx.Tx) error {
		var err error
		affected, err = sqlcgen.New(tx).UpdateTenant(ctx, sqlcgen.UpdateTenantParams{
			ID:     t.ID(),
			Name:   t.Name(),
			Status: string(t.Status()),
		})
		return err
	})
	switch {
	case err != nil:
		return fmt.Errorf("identity: update tenant: %w", err)
	case affected == 0:
		return domain.ErrTenantNotFound
	}
	return nil
}

func tenantFromRow(row sqlcgen.Tenant) (*domain.Tenant, error) {
	nit, err := domain.ParseNIT(row.Nit)
	if err != nil {
		return nil, fmt.Errorf("identity: tenant %s: stored NIT: %w", row.ID, err)
	}
	t, err := domain.RehydrateTenant(domain.TenantSnapshot{
		ID:        row.ID,
		Name:      row.Name,
		NIT:       nit,
		Status:    domain.TenantStatus(row.Status),
		CreatedAt: row.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("identity: tenant %s: %w", row.ID, err)
	}
	return t, nil
}
