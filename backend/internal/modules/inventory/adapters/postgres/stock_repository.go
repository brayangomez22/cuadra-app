package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
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

// Apply runs change on the locked levels and stores its result, all in one
// transaction:
//
//  1. creates the missing levels (empty), which checks that the product and
//     the locations exist in the tenant;
//  2. locks the levels with SELECT ... FOR UPDATE, so a concurrent movement
//     of the same product and location waits until this one commits and then
//     reads the new quantity;
//  3. calls change, which applies the domain rules (stock, average cost);
//  4. inserts the movements and updates the levels.
//
// Steps 1 and 2 go in location order, so two transfers in opposite
// directions take their locks in the same order and cannot deadlock.
func (r *StockRepository) Apply(ctx context.Context, tenantID, productID uuid.UUID, locationIDs []uuid.UUID, change domain.StockChange) error {
	sorted := slices.Clone(locationIDs)
	slices.SortFunc(sorted, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	if len(sorted) == 0 || len(slices.Compact(slices.Clone(sorted))) != len(sorted) {
		return fmt.Errorf("inventory: apply needs distinct locations, got %d", len(locationIDs))
	}

	// changeErr keeps the domain error apart from the database errors, which
	// are translated and wrapped.
	var changeErr error
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		now := r.now().UTC()
		for _, locationID := range sorted {
			if err := q.EnsureStockLevel(ctx, sqlcgen.EnsureStockLevelParams{
				TenantID: tenantID, ProductID: productID, LocationID: locationID, UpdatedAt: now,
			}); err != nil {
				return err
			}
		}
		rows, err := q.LockStockLevels(ctx, sqlcgen.LockStockLevelsParams{ProductID: productID, LocationIds: sorted})
		if err != nil {
			return err
		}
		levels, err := levelsInOrder(rows, locationIDs)
		if err != nil {
			return err
		}

		movements, err := change(levels)
		if err != nil {
			changeErr = err
			return err
		}

		for _, m := range movements {
			if err := q.InsertStockMovement(ctx, movementParams(m)); err != nil {
				return err
			}
		}
		for _, l := range levels {
			affected, err := q.UpdateStockLevel(ctx, sqlcgen.UpdateStockLevelParams{
				ProductID: l.ProductID(), LocationID: l.LocationID(),
				Quantity: l.Quantity(), AverageCost: l.AverageCost(), UpdatedAt: now,
			})
			if err != nil {
				return err
			}
			if affected != 1 {
				return fmt.Errorf("inventory: level of product %s at location %s vanished", l.ProductID(), l.LocationID())
			}
		}
		return nil
	})
	switch {
	case changeErr != nil:
		return changeErr
	case violates(err, foreignKeyViolation, stockLevelsProductFKey):
		return domain.ErrProductNotFound
	case violates(err, foreignKeyViolation, stockLevelsLocationFKey):
		return domain.ErrLocationNotFound
	case err != nil:
		return fmt.Errorf("inventory: apply stock change: %w", err)
	}
	return nil
}

// levelsInOrder rebuilds the locked levels in the order of locationIDs.
func levelsInOrder(rows []sqlcgen.LockStockLevelsRow, locationIDs []uuid.UUID) ([]*domain.StockLevel, error) {
	byLocation := make(map[uuid.UUID]*domain.StockLevel, len(rows))
	for _, row := range rows {
		l, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
			TenantID: row.TenantID, ProductID: row.ProductID, LocationID: row.LocationID,
			Quantity: row.Quantity, AverageCost: row.AverageCost,
		})
		if err != nil {
			return nil, fmt.Errorf("inventory: level of product %s at location %s: %w", row.ProductID, row.LocationID, err)
		}
		byLocation[row.LocationID] = l
	}
	levels := make([]*domain.StockLevel, 0, len(locationIDs))
	for _, id := range locationIDs {
		l, ok := byLocation[id]
		if !ok {
			return nil, fmt.Errorf("inventory: level at location %s not locked", id)
		}
		levels = append(levels, l)
	}
	return levels, nil
}

func movementParams(m *domain.StockMovement) sqlcgen.InsertStockMovementParams {
	p := sqlcgen.InsertStockMovementParams{
		ID:               m.ID(),
		TenantID:         m.TenantID(),
		ProductID:        m.ProductID(),
		LocationID:       m.LocationID(),
		Type:             m.Type().String(),
		Quantity:         m.Quantity(),
		UnitCost:         m.UnitCost(),
		BalanceAfter:     m.BalanceAfter(),
		AverageCostAfter: m.AverageCostAfter(),
		Reason:           m.Reason(),
		UserID:           m.UserID(),
		OccurredAt:       m.OccurredAt(),
	}
	if ref := m.Reference(); ref != nil {
		kind := string(ref.Kind)
		p.ReferenceKind, p.ReferenceID = &kind, &ref.ID
	}
	return p
}

// ListLevels returns one page of the levels at a location, by product, and
// how many match in total, read in the same transaction.
func (r *StockRepository) ListLevels(ctx context.Context, tenantID uuid.UUID, f domain.StockFilter) (domain.StockLevelPage, error) {
	limit, offset, err := pageBounds(f.Limit, f.Offset)
	if err != nil {
		return domain.StockLevelPage{}, err
	}
	var rows []sqlcgen.ListStockLevelsRow
	var total int64
	err = r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if rows, err = q.ListStockLevels(ctx, sqlcgen.ListStockLevelsParams{
			LocationID: f.LocationID, ProductID: f.ProductID, RowLimit: limit, RowOffset: offset,
		}); err != nil {
			return err
		}
		total, err = q.CountStockLevels(ctx, sqlcgen.CountStockLevelsParams{LocationID: f.LocationID, ProductID: f.ProductID})
		return err
	})
	if err != nil {
		return domain.StockLevelPage{}, fmt.Errorf("inventory: list stock levels: %w", err)
	}
	page := domain.StockLevelPage{Items: make([]*domain.StockLevel, 0, len(rows)), Total: int(total)}
	for _, row := range rows {
		l, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
			TenantID: row.TenantID, ProductID: row.ProductID, LocationID: row.LocationID,
			Quantity: row.Quantity, AverageCost: row.AverageCost,
		})
		if err != nil {
			return domain.StockLevelPage{}, fmt.Errorf("inventory: level of product %s at location %s: %w", row.ProductID, row.LocationID, err)
		}
		page.Items = append(page.Items, l)
	}
	return page, nil
}

// Kardex returns one page of a product's movements, newest first, and how
// many match in total, read in the same transaction.
func (r *StockRepository) Kardex(ctx context.Context, tenantID uuid.UUID, f domain.KardexFilter) (domain.MovementPage, error) {
	limit, offset, err := pageBounds(f.Limit, f.Offset)
	if err != nil {
		return domain.MovementPage{}, err
	}
	var rows []sqlcgen.StockMovement
	var total int64
	err = r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if rows, err = q.ListKardex(ctx, sqlcgen.ListKardexParams{
			ProductID: f.ProductID, LocationID: f.LocationID, RowLimit: limit, RowOffset: offset,
		}); err != nil {
			return err
		}
		total, err = q.CountKardex(ctx, sqlcgen.CountKardexParams{ProductID: f.ProductID, LocationID: f.LocationID})
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
		ProductID:        row.ProductID,
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
