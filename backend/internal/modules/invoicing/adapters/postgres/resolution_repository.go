package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// ResolutionRepository implements domain.ResolutionRepository.
type ResolutionRepository struct{ db *db.DB }

// NewResolutionRepository returns a ResolutionRepository on d.
func NewResolutionRepository(d *db.DB) *ResolutionRepository { return &ResolutionRepository{db: d} }

// Create stores a new resolution.
func (r *ResolutionRepository) Create(ctx context.Context, res *domain.Resolution) error {
	s := res.Snapshot()
	p := s.Params
	err := r.db.WithTenantTx(ctx, s.TenantID, func(tx pgx.Tx) error {
		return sqlcgen.New(tx).CreateResolution(ctx, sqlcgen.CreateResolutionParams{
			ID: s.ID, TenantID: s.TenantID, Kind: string(p.Kind), Prefix: p.Prefix, Number: p.Number,
			TechnicalKey: p.TechnicalKey, RangeFrom: p.From, RangeTo: p.To,
			ValidFrom: p.ValidFrom.Time(), ValidTo: p.ValidTo.Time(), NextNumber: s.Next, CreatedAt: s.CreatedAt,
		})
	})
	if err != nil {
		return fmt.Errorf("invoicing: create resolution: %w", err)
	}
	return nil
}

// List returns the tenant's resolutions, the newest first.
func (r *ResolutionRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.Resolution, error) {
	rows, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]sqlcgen.NumberingResolution, error) {
		return sqlcgen.New(tx).ListResolutions(ctx, tenantID)
	})
	if err != nil {
		return nil, fmt.Errorf("invoicing: list resolutions: %w", err)
	}
	return resolutionsOf(rows), nil
}

// Delete removes a resolution that numbered nothing.
func (r *ResolutionRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	exists, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (bool, error) {
		q := sqlcgen.New(tx)
		n, err := q.DeleteUnusedResolution(ctx, sqlcgen.DeleteUnusedResolutionParams{TenantID: tenantID, ID: id})
		if err != nil || n > 0 {
			return true, err
		}
		found, err := q.ResolutionExists(ctx, sqlcgen.ResolutionExistsParams{TenantID: tenantID, ID: id})
		if err == nil && found {
			return true, domain.ErrResolutionInUse
		}
		return false, err
	})
	switch {
	case err != nil:
		return fmt.Errorf("invoicing: delete resolution: %w", err)
	case !exists:
		return domain.ErrResolutionNotFound
	}
	return nil
}

// LockKind returns the tenant's resolutions of kind, locked FOR UPDATE until
// the transaction of ctx ends: the sale's, when it joins one.
func (r *ResolutionRepository) LockKind(ctx context.Context, tenantID uuid.UUID, kind domain.Kind) ([]*domain.Resolution, error) {
	rows, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]sqlcgen.NumberingResolution, error) {
		return sqlcgen.New(tx).LockResolutionsOfKind(ctx, sqlcgen.LockResolutionsOfKindParams{TenantID: tenantID, Kind: string(kind)})
	})
	if err != nil {
		return nil, fmt.Errorf("invoicing: lock resolutions: %w", err)
	}
	return resolutionsOf(rows), nil
}

// SaveNext stores the resolution's next consecutive.
func (r *ResolutionRepository) SaveNext(ctx context.Context, res *domain.Resolution) error {
	n, err := db.Affected(ctx, r.db, res.TenantID(), func(tx pgx.Tx) (int64, error) {
		return sqlcgen.New(tx).SaveResolutionNext(ctx, sqlcgen.SaveResolutionNextParams{
			TenantID: res.TenantID(), ID: res.ID(), NextNumber: res.Next(),
		})
	})
	switch {
	case err != nil:
		return fmt.Errorf("invoicing: save next number: %w", err)
	case n == 0:
		return domain.ErrResolutionNotFound
	}
	return nil
}

func resolutionsOf(rows []sqlcgen.NumberingResolution) []*domain.Resolution {
	out := make([]*domain.Resolution, len(rows))
	for i, row := range rows {
		out[i] = domain.RestoreResolution(domain.ResolutionSnapshot{
			ID: row.ID, TenantID: row.TenantID, Next: row.NextNumber, CreatedAt: row.CreatedAt.UTC(),
			Params: domain.ResolutionParams{
				Kind: domain.Kind(row.Kind), Prefix: row.Prefix, Number: row.Number, TechnicalKey: row.TechnicalKey,
				From: row.RangeFrom, To: row.RangeTo,
				ValidFrom: domain.DateOf(row.ValidFrom), ValidTo: domain.DateOf(row.ValidTo),
			},
		})
	}
	return out
}
