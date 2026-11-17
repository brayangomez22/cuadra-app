// Package postgres keeps the feature flags: the global values in
// feature_flags (platform configuration, without RLS) and the exceptions of
// each tenant in tenant_feature_flags (with Row-Level Security).
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags/adapters/postgres/sqlcgen"
)

// exceptionsQuery calls the SECURITY DEFINER function of migration 00036.
// It is written by hand: sqlc cannot type the columns of a function
// returning TABLE.
const exceptionsQuery = `SELECT tenant_id, tenant_name, key, enabled FROM feature_flag_exceptions()`

// Store implements flags.Store.
type Store struct{ db *db.DB }

var _ flags.Store = (*Store)(nil)

// NewStore builds a Store on d.
func NewStore(d *db.DB) *Store { return &Store{db: d} }

// Globals returns the global value of each flag that has one. They are not
// a tenant's data, so they are read outside a tenant transaction.
func (s *Store) Globals(ctx context.Context) (map[string]bool, error) {
	out := map[string]bool{}
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		rows, err := sqlcgen.New(tx).ListGlobalFlags(ctx)
		for _, r := range rows {
			out[r.Key] = r.Enabled
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("flags: list global values: %w", err)
	}
	return out, nil
}

// Exceptions returns the tenant's exceptions, by key.
func (s *Store) Exceptions(ctx context.Context, tenantID uuid.UUID) (map[string]bool, error) {
	out := map[string]bool{}
	err := s.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := sqlcgen.New(tx).ListTenantFlags(ctx, tenantID)
		for _, r := range rows {
			out[r.Key] = r.Enabled
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("flags: list exceptions: %w", err)
	}
	return out, nil
}

// AllExceptions lists the exceptions of every tenant, with the tenant's
// name, through flag_lookup (it reads no other column of the tenants).
func (s *Store) AllExceptions(ctx context.Context) ([]flags.Exception, error) {
	var out []flags.Exception
	// Not a tenant transaction: it lists every tenant's exceptions.
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, exceptionsQuery)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (flags.Exception, error) {
			var ex flags.Exception
			err := row.Scan(&ex.TenantID, &ex.TenantName, &ex.Key, &ex.Enabled)
			return ex, err
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("flags: list all exceptions: %w", err)
	}
	return out, nil
}

// SetGlobal stores the global value of a flag.
func (s *Store) SetGlobal(ctx context.Context, c flags.Change) error {
	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		return sqlcgen.New(tx).SetGlobalFlag(ctx, sqlcgen.SetGlobalFlagParams{
			Key: c.Key, Enabled: c.Enabled, UpdatedAt: c.At, UpdatedBy: c.By,
		})
	})
	if err != nil {
		return fmt.Errorf("flags: set %s: %w", c.Key, err)
	}
	return nil
}

// SetException stores a tenant's exception, in a transaction of that
// tenant: the operator chose it, it does not come from a request body of the
// tenant's own users. A tenant that does not exist is ErrUnknownTenant.
func (s *Store) SetException(ctx context.Context, tenantID uuid.UUID, c flags.Change) error {
	err := s.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return sqlcgen.New(tx).SetTenantFlag(ctx, sqlcgen.SetTenantFlagParams{
			TenantID: tenantID, Key: c.Key, Enabled: c.Enabled, UpdatedAt: c.At, UpdatedBy: c.By,
		})
	})
	switch {
	case db.Violates(err, db.ForeignKeyViolation, "tenant_feature_flags_tenant_id_fkey"):
		return flags.ErrUnknownTenant
	case err != nil:
		return fmt.Errorf("flags: set %s for a tenant: %w", c.Key, err)
	}
	return nil
}

// ClearException removes a tenant's exception, if it has one.
func (s *Store) ClearException(ctx context.Context, tenantID uuid.UUID, key string) error {
	err := s.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return sqlcgen.New(tx).ClearTenantFlag(ctx, sqlcgen.ClearTenantFlagParams{TenantID: tenantID, Key: key})
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("flags: clear %s for a tenant: %w", key, err)
	}
	return nil
}
