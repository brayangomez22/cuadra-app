// Package postgres keeps the users' comments in PostgreSQL, with Row-Level
// Security.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/feedback/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Enqueue queues the notice of a comment in the transaction that stores it.
type Enqueue func(ctx context.Context, tx pgx.Tx, tenantID, feedbackID uuid.UUID) error

// FeedbackRepository implements domain.Repository.
type FeedbackRepository struct {
	db      *db.DB
	enqueue Enqueue
}

var _ domain.Repository = (*FeedbackRepository)(nil)

// NewFeedbackRepository builds the repository on d; enqueue queues each
// comment's notice.
func NewFeedbackRepository(d *db.DB, enqueue Enqueue) *FeedbackRepository {
	return &FeedbackRepository{db: d, enqueue: enqueue}
}

// Create stores the comment and queues its notice in the same transaction
// (transactional outbox): if either fails, neither happens.
func (r *FeedbackRepository) Create(ctx context.Context, f *domain.Feedback) error {
	err := r.db.WithTenantTx(ctx, f.TenantID(), func(tx pgx.Tx) error {
		err := sqlcgen.New(tx).CreateFeedback(ctx, sqlcgen.CreateFeedbackParams{
			ID: f.ID(), TenantID: f.TenantID(), UserID: f.UserID(), Message: f.Message(),
			Screen: f.Screen(), AppVersion: f.AppVersion(), CreatedAt: f.CreatedAt(),
		})
		if err != nil {
			return err
		}
		return r.enqueue(ctx, tx, f.TenantID(), f.ID())
	})
	if err != nil {
		return fmt.Errorf("feedback: create: %w", err)
	}
	return nil
}

// Get returns a comment of the tenant.
func (r *FeedbackRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.Feedback, error) {
	row, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (sqlcgen.Feedback, error) {
		return sqlcgen.New(tx).GetFeedback(ctx, sqlcgen.GetFeedbackParams{TenantID: tenantID, ID: id})
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrFeedbackMissing
	case err != nil:
		return nil, fmt.Errorf("feedback: get: %w", err)
	}
	f, err := domain.RehydrateFeedback(domain.FeedbackSnapshot{
		ID: row.ID, TenantID: row.TenantID, UserID: row.UserID, Message: row.Message, Screen: row.Screen,
		AppVersion: row.AppVersion, CreatedAt: row.CreatedAt, NotifiedAt: row.NotifiedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("feedback: stored comment %s: %w", row.ID, err)
	}
	return f, nil
}

// MarkNotified stores when the team was told about f.
func (r *FeedbackRepository) MarkNotified(ctx context.Context, f *domain.Feedback) error {
	err := r.db.WithTenantTx(ctx, f.TenantID(), func(tx pgx.Tx) error {
		return sqlcgen.New(tx).MarkFeedbackNotified(ctx, sqlcgen.MarkFeedbackNotifiedParams{
			TenantID: f.TenantID(), ID: f.ID(), NotifiedAt: f.NotifiedAt(),
		})
	})
	if err != nil {
		return fmt.Errorf("feedback: mark notified: %w", err)
	}
	return nil
}
