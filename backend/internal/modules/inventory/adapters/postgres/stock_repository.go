package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// StockRepository implements domain.StockRepository.
type StockRepository struct {
	db  *db.DB
	now func() time.Time
}

// NewStockRepository returns a StockRepository on d.
func NewStockRepository(d *db.DB) *StockRepository { return &StockRepository{db: d, now: time.Now} }

// ListLevels returns one page of the levels without a lot at a location, by
// variant, and how many match in total, read in the same transaction.
func (r *StockRepository) ListLevels(ctx context.Context, tenantID uuid.UUID, f domain.StockFilter) (domain.StockLevelPage, error) {
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
	if err != nil {
		return domain.StockLevelPage{}, err
	}
	var rows []sqlcgen.ListStockLevelsRow
	var total int64
	err = r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if rows, err = q.ListStockLevels(ctx, sqlcgen.ListStockLevelsParams{
			LocationID: f.LocationID, VariantID: f.VariantID, RowLimit: limit, RowOffset: offset,
		}); err != nil {
			return err
		}
		total, err = q.CountStockLevels(ctx, sqlcgen.CountStockLevelsParams{LocationID: f.LocationID, VariantID: f.VariantID})
		return err
	})
	if err != nil {
		return domain.StockLevelPage{}, fmt.Errorf("inventory: list stock levels: %w", err)
	}
	page := domain.StockLevelPage{Items: make([]*domain.StockLevel, 0, len(rows)), Total: int(total)}
	for _, row := range rows {
		l, err := rehydrateLevel(domain.StockLevelSnapshot{
			TenantID: row.TenantID, VariantID: row.VariantID, LocationID: row.LocationID,
			Quantity: row.Quantity, AverageCost: row.AverageCost,
		})
		if err != nil {
			return domain.StockLevelPage{}, err
		}
		page.Items = append(page.Items, l)
	}
	return page, nil
}

// Kardex returns one page of a variant's movements, newest first, and how
// many match in total, read in the same transaction.
func (r *StockRepository) Kardex(ctx context.Context, tenantID uuid.UUID, f domain.KardexFilter) (domain.MovementPage, error) {
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
	if err != nil {
		return domain.MovementPage{}, err
	}
	var rows []sqlcgen.StockMovement
	var total int64
	err = r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if rows, err = q.ListKardex(ctx, sqlcgen.ListKardexParams{
			VariantID: f.VariantID, LocationID: f.LocationID, RowLimit: limit, RowOffset: offset,
		}); err != nil {
			return err
		}
		total, err = q.CountKardex(ctx, sqlcgen.CountKardexParams{VariantID: f.VariantID, LocationID: f.LocationID})
		return err
	})
	if err != nil {
		return domain.MovementPage{}, fmt.Errorf("inventory: kardex: %w", err)
	}
	page := domain.MovementPage{Items: make([]*domain.StockMovement, 0, len(rows)), Total: int(total)}
	for _, row := range rows {
		m, err := movementFromRow(row)
		if err != nil {
			return domain.MovementPage{}, err
		}
		page.Items = append(page.Items, m)
	}
	return page, nil
}

func movementFromRow(row sqlcgen.StockMovement) (*domain.StockMovement, error) {
	t, err := domain.ParseMovementType(row.Type)
	if err != nil {
		return nil, fmt.Errorf("inventory: movement %s: stored type: %w", row.ID, err)
	}
	var ref *domain.DocumentRef
	switch {
	case row.ReferenceKind != nil && row.ReferenceID != nil:
		ref = &domain.DocumentRef{Kind: domain.DocumentKind(*row.ReferenceKind), ID: *row.ReferenceID}
	case row.ReferenceKind != nil || row.ReferenceID != nil:
		return nil, fmt.Errorf("inventory: movement %s: %w", row.ID, errors.New("half a document reference"))
	}
	m, err := domain.RehydrateStockMovement(domain.StockMovementSnapshot{
		ID:               row.ID,
		TenantID:         row.TenantID,
		VariantID:        row.VariantID,
		LocationID:       row.LocationID,
		Type:             t,
		Quantity:         row.Quantity,
		UnitCost:         row.UnitCost,
		BalanceAfter:     row.BalanceAfter,
		AverageCostAfter: row.AverageCostAfter,
		Reason:           row.Reason,
		Reference:        ref,
		UserID:           row.UserID,
		OccurredAt:       row.OccurredAt,
	})
	if err != nil {
		return nil, fmt.Errorf("inventory: movement %s: %w", row.ID, err)
	}
	return m, nil
}

// MovementsByReference returns the movements of a document, oldest first.
func (r *StockRepository) MovementsByReference(ctx context.Context, tenantID uuid.UUID, ref domain.DocumentRef) ([]*domain.StockMovement, error) {
	rows, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]sqlcgen.StockMovement, error) {
		return sqlcgen.New(tx).ListMovementsByReference(ctx, sqlcgen.ListMovementsByReferenceParams{
			ReferenceKind: string(ref.Kind), ReferenceID: ref.ID,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("inventory: movements of %s %s: %w", ref.Kind, ref.ID, err)
	}
	out := make([]*domain.StockMovement, 0, len(rows))
	for _, row := range rows {
		m, err := movementFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// rehydrateLevel rebuilds a stored level.
func rehydrateLevel(s domain.StockLevelSnapshot) (*domain.StockLevel, error) {
	l, err := domain.RehydrateStockLevel(s)
	if err != nil {
		return nil, fmt.Errorf("inventory: level of variant %s at location %s: %w", s.VariantID, s.LocationID, err)
	}
	return l, nil
}
