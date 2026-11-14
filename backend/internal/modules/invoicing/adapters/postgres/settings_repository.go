// Package postgres stores the invoicing module on PostgreSQL. Every
// operation runs in db.WithTenantTx, so Row-Level Security limits it to one
// tenant; with a ctx from db.InTenantTx (the sale's) it joins that
// transaction.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// SettingsRepository implements domain.SettingsRepository.
type SettingsRepository struct{ db *db.DB }

// NewSettingsRepository returns a SettingsRepository on d.
func NewSettingsRepository(d *db.DB) *SettingsRepository { return &SettingsRepository{db: d} }

// Get returns the tenant's settings, or the defaults if it has none.
func (r *SettingsRepository) Get(ctx context.Context, tenantID uuid.UUID) (domain.Settings, error) {
	row, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (sqlcgen.GetInvoicingSettingsRow, error) {
		return sqlcgen.New(tx).GetInvoicingSettings(ctx, tenantID)
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.DefaultSettings(), nil
	case err != nil:
		return domain.Settings{}, fmt.Errorf("invoicing: get settings: %w", err)
	}
	return domain.Settings{Enabled: row.Enabled, DefaultKind: domain.Kind(row.DefaultKind)}, nil
}

// Save stores the tenant's settings.
func (r *SettingsRepository) Save(ctx context.Context, tenantID uuid.UUID, s domain.Settings) error {
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return sqlcgen.New(tx).SaveInvoicingSettings(ctx, sqlcgen.SaveInvoicingSettingsParams{
			TenantID: tenantID, Enabled: s.Enabled, DefaultKind: string(s.DefaultKind),
		})
	})
	if err != nil {
		return fmt.Errorf("invoicing: save settings: %w", err)
	}
	return nil
}
