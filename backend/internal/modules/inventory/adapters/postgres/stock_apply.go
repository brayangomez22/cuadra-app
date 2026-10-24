package postgres

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Apply runs change on the locked levels and stores its result, all in one
// transaction:
//
//  1. creates the missing levels (empty, without a lot), which checks that
//     the variant and the locations exist in the tenant;
//  2. locks the levels with SELECT ... FOR UPDATE, so a concurrent movement
//     of the same variant and location waits until this one commits and then
//     reads the new quantity;
//  3. calls change, which applies the domain rules (stock, average cost);
//  4. inserts the movements and updates the levels.
//
// Steps 1 and 2 go in location order, so two transfers in opposite
// directions take their locks in the same order and cannot deadlock. Callers
// that apply several variants in one transaction (a sale) go in variant
// order, so every lock is taken in (variant, location) order.
func (r *StockRepository) Apply(ctx context.Context, tenantID, variantID uuid.UUID, locationIDs []uuid.UUID, change domain.StockChange) error {
	sorted, err := lockOrder(locationIDs)
	if err != nil {
		return err
	}
	// changeErr keeps the domain error apart from the database errors, which
	// are translated and wrapped.
	var changeErr error
	err = r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		t := stockTx{q: sqlcgen.New(tx), tenantID: tenantID, variantID: variantID, now: r.now().UTC()}
		levels, err := t.lockLevels(ctx, sorted, locationIDs)
		if err != nil {
			return err
		}
		movements, err := change(levels)
		if err != nil {
			changeErr = err
			return err
		}
		return t.save(ctx, movements, levels)
	})
	if changeErr != nil {
		return changeErr
	}
	return applyError(err)
}

// lockOrder returns locationIDs in the order their levels are locked; they
// must be distinct.
func lockOrder(locationIDs []uuid.UUID) ([]uuid.UUID, error) {
	sorted := slices.Clone(locationIDs)
	slices.SortFunc(sorted, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	if len(sorted) == 0 || len(slices.Compact(slices.Clone(sorted))) != len(sorted) {
		return nil, fmt.Errorf("inventory: apply needs distinct locations, got %d", len(locationIDs))
	}
	return sorted, nil
}

// applyError translates the database errors of Apply.
func applyError(err error) error {
	switch {
	case db.Violates(err, db.ForeignKeyViolation, stockLevelsVariantFKey):
		return domain.ErrVariantNotFound
	case db.Violates(err, db.ForeignKeyViolation, stockLevelsLocationFKey):
		return domain.ErrLocationNotFound
	case err != nil:
		return fmt.Errorf("inventory: apply stock change: %w", err)
	}
	return nil
}

// stockTx is a stock change of one variant inside its transaction.
type stockTx struct {
	q                   *sqlcgen.Queries
	tenantID, variantID uuid.UUID
	now                 time.Time
}

// lockLevels creates the missing levels at the sorted locations (step 1)
// and locks them (step 2), both in that order. It returns them in the order
// of locationIDs.
func (t stockTx) lockLevels(ctx context.Context, sorted, locationIDs []uuid.UUID) ([]*domain.StockLevel, error) {
	for _, locationID := range sorted {
		if err := t.q.EnsureStockLevel(ctx, sqlcgen.EnsureStockLevelParams{
			TenantID: t.tenantID, VariantID: t.variantID, LocationID: locationID, UpdatedAt: t.now,
		}); err != nil {
			return nil, err
		}
	}
	rows, err := t.q.LockStockLevels(ctx, sqlcgen.LockStockLevelsParams{VariantID: t.variantID, LocationIds: sorted})
	if err != nil {
		return nil, err
	}
	return levelsInOrder(rows, locationIDs)
}

// save inserts the movements and updates the levels (step 4).
func (t stockTx) save(ctx context.Context, movements []*domain.StockMovement, levels []*domain.StockLevel) error {
	for _, m := range movements {
		if err := t.q.InsertStockMovement(ctx, movementParams(m)); err != nil {
			return err
		}
	}
	for _, l := range levels {
		affected, err := t.q.UpdateStockLevel(ctx, sqlcgen.UpdateStockLevelParams{
			VariantID: l.VariantID(), LocationID: l.LocationID(),
			Quantity: l.Quantity(), AverageCost: l.AverageCost(), UpdatedAt: t.now,
		})
		if err != nil {
			return err
		}
		if affected != 1 {
			return fmt.Errorf("inventory: level of variant %s at location %s vanished", l.VariantID(), l.LocationID())
		}
	}
	return nil
}

// levelsInOrder rebuilds the locked levels in the order of locationIDs.
func levelsInOrder(rows []sqlcgen.LockStockLevelsRow, locationIDs []uuid.UUID) ([]*domain.StockLevel, error) {
	byLocation := make(map[uuid.UUID]*domain.StockLevel, len(rows))
	for _, row := range rows {
		l, err := rehydrateLevel(domain.StockLevelSnapshot{
			TenantID: row.TenantID, VariantID: row.VariantID, LocationID: row.LocationID,
			Quantity: row.Quantity, AverageCost: row.AverageCost,
		})
		if err != nil {
			return nil, err
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
		VariantID:        m.VariantID(),
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
