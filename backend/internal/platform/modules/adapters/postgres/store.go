// Package postgres stores the active modules of each tenant in
// tenant_modules, with Row-Level Security.
package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules/adapters/postgres/sqlcgen"
)

// Store implements modules.Store.
type Store struct{ db *db.DB }

var _ modules.Store = (*Store)(nil)

// NewStore builds a Store on d.
func NewStore(d *db.DB) *Store { return &Store{db: d} }

// Active lists the module IDs the tenant has active.
func (s *Store) Active(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	var ids []string
	err := s.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		ids, err = sqlcgen.New(tx).ListActiveModules(ctx, tenantID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("modules: list active modules: %w", err)
	}
	return ids, nil
}

// Update applies decide's change in one transaction. An advisory lock per
// tenant serializes the updates of one tenant: the rows decide reads cannot
// be locked with FOR UPDATE, because the module it checks may have no row.
func (s *Store) Update(ctx context.Context, tenantID uuid.UUID, decide func(active []string) (*modules.Change, error)) error {
	return s.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := q.LockTenantModules(ctx, tenantID); err != nil {
			return fmt.Errorf("modules: lock tenant modules: %w", err)
		}
		active, err := q.ListActiveModules(ctx, tenantID)
		if err != nil {
			return fmt.Errorf("modules: list active modules: %w", err)
		}
		change, err := decide(active)
		if err != nil || change == nil {
			return err
		}
		if !change.Active {
			if err := q.DisableModule(ctx, sqlcgen.DisableModuleParams{TenantID: tenantID, ModuleID: change.Module}); err != nil {
				return fmt.Errorf("modules: disable %s: %w", change.Module, err)
			}
			return nil
		}
		var by *uuid.UUID
		if change.By != uuid.Nil {
			by = &change.By
		}
		err = q.EnableModule(ctx, sqlcgen.EnableModuleParams{
			TenantID: tenantID, ModuleID: change.Module, EnabledAt: change.At, EnabledBy: by,
		})
		if err != nil {
			return fmt.Errorf("modules: enable %s: %w", change.Module, err)
		}
		return nil
	})
}
