// Package postgres implements the customers repositories on PostgreSQL. Every
// query runs inside db.WithTenantTx, so Row-Level Security scopes it to the
// tenant.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Constraints the repositories translate to domain errors (00024_customers).
const (
	segmentNameKey     = "customer_segments_tenant_id_name_key"
	customerDocKey     = "customers_document_key"
	customerSegmentKey = "customers_segment_fkey"
)

// SegmentRepository implements domain.SegmentRepository.
type SegmentRepository struct{ db *db.DB }

// NewSegmentRepository returns a SegmentRepository on d.
func NewSegmentRepository(d *db.DB) *SegmentRepository { return &SegmentRepository{db: d} }

// Create stores a new segment; domain.ErrSegmentNameTaken if another segment
// of the tenant has its name, ignoring case.
func (r *SegmentRepository) Create(ctx context.Context, s *domain.Segment) error {
	err := r.db.WithTenantTx(ctx, s.TenantID(), func(tx pgx.Tx) error {
		return sqlcgen.New(tx).CreateSegment(ctx, sqlcgen.CreateSegmentParams{
			ID: s.ID(), TenantID: s.TenantID(), Name: s.Name(), CreatedAt: s.CreatedAt(),
		})
	})
	return segmentWriteError("create", err)
}

// Update stores the segment's name.
func (r *SegmentRepository) Update(ctx context.Context, s *domain.Segment) error {
	affected, err := db.Affected(ctx, r.db, s.TenantID(), func(tx pgx.Tx) (int64, error) {
		return sqlcgen.New(tx).RenameSegment(ctx, sqlcgen.RenameSegmentParams{ID: s.ID(), Name: s.Name()})
	})
	if err == nil && affected == 0 {
		return domain.ErrSegmentNotFound
	}
	return segmentWriteError("update", err)
}

func segmentWriteError(op string, err error) error {
	switch {
	case db.Violates(err, db.UniqueViolation, segmentNameKey):
		return domain.ErrSegmentNameTaken
	case err != nil:
		return fmt.Errorf("customers: %s segment: %w", op, err)
	}
	return nil
}

// GetByID returns domain.ErrSegmentNotFound when the segment does not exist
// in tenantID.
func (r *SegmentRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Segment, error) {
	row, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (sqlcgen.CustomerSegment, error) {
		return sqlcgen.New(tx).GetSegment(ctx, id)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSegmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("customers: get segment: %w", err)
	}
	return segmentFromRow(row)
}

// List returns every segment of the tenant, by name.
func (r *SegmentRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Segment, error) {
	rows, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]sqlcgen.CustomerSegment, error) {
		return sqlcgen.New(tx).ListSegments(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("customers: list segments: %w", err)
	}
	out := make([]*domain.Segment, 0, len(rows))
	for _, row := range rows {
		s, err := segmentFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// Delete removes the segment; domain.ErrSegmentInUse if a customer, active
// or not, references it (the foreign key of customers rejects the delete).
func (r *SegmentRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	affected, err := db.Affected(ctx, r.db, tenantID, func(tx pgx.Tx) (int64, error) {
		return sqlcgen.New(tx).DeleteSegment(ctx, id)
	})
	switch {
	case db.Violates(err, db.ForeignKeyViolation, customerSegmentKey):
		return domain.ErrSegmentInUse
	case err != nil:
		return fmt.Errorf("customers: delete segment: %w", err)
	case affected == 0:
		return domain.ErrSegmentNotFound
	}
	return nil
}

func segmentFromRow(row sqlcgen.CustomerSegment) (*domain.Segment, error) {
	s, err := domain.RehydrateSegment(domain.SegmentSnapshot{
		ID: row.ID, TenantID: row.TenantID, Name: row.Name, CreatedAt: row.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("customers: segment %s: %w", row.ID, err)
	}
	return s, nil
}
