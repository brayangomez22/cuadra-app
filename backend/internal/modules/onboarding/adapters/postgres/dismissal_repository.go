// Package postgres reads what the getting-started checklist needs from the
// other modules' tables and keeps who hid it, in PostgreSQL with Row-Level
// Security.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// DismissalRepository implements domain.DismissalRepository.
type DismissalRepository struct{ db *db.DB }

var _ domain.DismissalRepository = (*DismissalRepository)(nil)

// NewDismissalRepository builds the repository on d.
func NewDismissalRepository(d *db.DB) *DismissalRepository { return &DismissalRepository{db: d} }

// Get returns the user's dismissal, if they hid the checklist.
func (r *DismissalRepository) Get(ctx context.Context, tenantID, userID uuid.UUID) (domain.Dismissal, bool, error) {
	row, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (sqlcgen.GetDismissalRow, error) {
		return sqlcgen.New(tx).GetDismissal(ctx, sqlcgen.GetDismissalParams{TenantID: tenantID, UserID: userID})
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Dismissal{}, false, nil
	case err != nil:
		return domain.Dismissal{}, false, fmt.Errorf("onboarding: get dismissal: %w", err)
	}
	pending := make([]domain.StepID, len(row.PendingSteps))
	for i, s := range row.PendingSteps {
		pending[i] = domain.StepID(s)
	}
	return domain.RestoreDismissal(tenantID, userID, row.DismissedAt, pending), true, nil
}

// Save stores d, replacing the user's earlier dismissal.
func (r *DismissalRepository) Save(ctx context.Context, d domain.Dismissal) error {
	pending := make([]string, 0, len(d.Pending()))
	for _, s := range d.Pending() {
		pending = append(pending, string(s))
	}
	err := r.db.WithTenantTx(ctx, d.TenantID(), func(tx pgx.Tx) error {
		return sqlcgen.New(tx).SaveDismissal(ctx, sqlcgen.SaveDismissalParams{
			TenantID: d.TenantID(), UserID: d.UserID(), DismissedAt: d.DismissedAt(), PendingSteps: pending,
		})
	})
	if err != nil {
		return fmt.Errorf("onboarding: save dismissal: %w", err)
	}
	return nil
}
