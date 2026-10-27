package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// ReorderRepository implements domain.ReorderRuleRepository on reorder_rules.
type ReorderRepository struct {
	db  *db.DB
	now func() time.Time
}

// NewReorderRepository returns a ReorderRepository on d.
func NewReorderRepository(d *db.DB) *ReorderRepository {
	return &ReorderRepository{db: d, now: time.Now}
}

// Save inserts the rule or replaces the one of its variant at its location.
// The foreign keys report a variant or location missing in the tenant.
func (r *ReorderRepository) Save(ctx context.Context, rule *domain.ReorderRule) error {
	err := r.db.WithTenantTx(ctx, rule.TenantID(), func(tx pgx.Tx) error {
		return sqlcgen.New(tx).UpsertReorderRule(ctx, sqlcgen.UpsertReorderRuleParams{
			TenantID: rule.TenantID(), VariantID: rule.VariantID(), LocationID: rule.LocationID(),
			MinQuantity: rule.Min(), MaxQuantity: rule.Max(), UpdatedAt: r.now().UTC(),
		})
	})
	switch {
	case db.Violates(err, db.ForeignKeyViolation, reorderRulesVariantFKey):
		return domain.ErrVariantNotFound
	case db.Violates(err, db.ForeignKeyViolation, reorderRulesLocationFKey):
		return domain.ErrLocationNotFound
	case err != nil:
		return fmt.Errorf("inventory: save reorder rule: %w", err)
	}
	return nil
}

// Delete removes the rule of variantID at locationID.
func (r *ReorderRepository) Delete(ctx context.Context, tenantID, variantID, locationID uuid.UUID) error {
	n, err := db.Affected(ctx, r.db, tenantID, func(tx pgx.Tx) (int64, error) {
		return sqlcgen.New(tx).DeleteReorderRule(ctx, sqlcgen.DeleteReorderRuleParams{VariantID: variantID, LocationID: locationID})
	})
	switch {
	case err != nil:
		return fmt.Errorf("inventory: delete reorder rule: %w", err)
	case n == 0:
		return domain.ErrReorderRuleNotFound
	}
	return nil
}

// ForVariants returns the rules at locationID of variantIDs, by variant.
func (r *ReorderRepository) ForVariants(ctx context.Context, tenantID, locationID uuid.UUID, variantIDs []uuid.UUID) (map[uuid.UUID]*domain.ReorderRule, error) {
	rows, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]sqlcgen.ListReorderRulesForVariantsRow, error) {
		return sqlcgen.New(tx).ListReorderRulesForVariants(ctx, sqlcgen.ListReorderRulesForVariantsParams{
			LocationID: locationID, VariantIds: variantIDs,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("inventory: list reorder rules: %w", err)
	}
	out := make(map[uuid.UUID]*domain.ReorderRule, len(rows))
	for _, row := range rows {
		rule, err := rehydrateRule(domain.ReorderRuleSpec{
			TenantID: row.TenantID, VariantID: row.VariantID, LocationID: row.LocationID,
			Min: row.MinQuantity, Max: row.MaxQuantity,
		})
		if err != nil {
			return nil, err
		}
		out[row.VariantID] = rule
	}
	return out, nil
}

// Alerts returns one page of the rules at or below their minimum and how many
// match in total, read in the same transaction.
func (r *ReorderRepository) Alerts(ctx context.Context, tenantID uuid.UUID, f domain.ReorderFilter) (domain.RuleStockPage, error) {
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
	if err != nil {
		return domain.RuleStockPage{}, err
	}
	var rows []sqlcgen.ListReorderAlertsRow
	var total int64
	err = r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if rows, err = q.ListReorderAlerts(ctx, sqlcgen.ListReorderAlertsParams{
			LocationID: f.LocationID, RowLimit: limit, RowOffset: offset,
		}); err != nil {
			return err
		}
		total, err = q.CountReorderAlerts(ctx, f.LocationID)
		return err
	})
	if err != nil {
		return domain.RuleStockPage{}, fmt.Errorf("inventory: list reorder alerts: %w", err)
	}
	page := domain.RuleStockPage{Items: make([]domain.RuleStock, 0, len(rows)), Total: int(total)}
	for _, row := range rows {
		rule, err := rehydrateRule(domain.ReorderRuleSpec{
			TenantID: row.TenantID, VariantID: row.VariantID, LocationID: row.LocationID,
			Min: row.MinQuantity, Max: row.MaxQuantity,
		})
		if err != nil {
			return domain.RuleStockPage{}, err
		}
		page.Items = append(page.Items, domain.RuleStock{Rule: rule, OnHand: row.OnHand})
	}
	return page, nil
}

// rehydrateRule rebuilds a stored rule; a row the domain rejects is corrupt.
func rehydrateRule(s domain.ReorderRuleSpec) (*domain.ReorderRule, error) {
	rule, err := domain.NewReorderRule(s)
	if err != nil {
		return nil, fmt.Errorf("inventory: stored reorder rule is invalid: %w", err)
	}
	return rule, nil
}
