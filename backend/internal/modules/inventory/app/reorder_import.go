package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ReorderLine is the minimum and maximum stock of a variant, in its base
// unit.
type ReorderLine struct {
	VariantID uuid.UUID
	Min       decimal.Decimal
	Max       decimal.Decimal
}

// SetReorderRules stores the minimum and maximum of several variants at an
// active location, replacing the previous ones, as an import loads them.
// Every rule is checked before any is saved (ErrInvalidReorderRange). With
// a ctx from db.InTenantTx it joins the import's batch, so the rules are
// written with the products they belong to.
func (s *Service) SetReorderRules(ctx context.Context, a Actor, locationID uuid.UUID, lines []ReorderLine) (err error) {
	ctx, span := s.startSpan(ctx, "SetReorderRules", a, locationAttr(locationID), attribute.Int("lines", len(lines)))
	defer func() { usecase.End(span, err) }()

	rules := make([]*domain.ReorderRule, 0, len(lines))
	for _, l := range lines {
		rule, err := domain.NewReorderRule(domain.ReorderRuleSpec{
			TenantID: a.TenantID, VariantID: l.VariantID, LocationID: locationID, Min: l.Min, Max: l.Max,
		})
		if err != nil {
			return err
		}
		rules = append(rules, rule)
	}
	if err := s.requireActiveLocation(ctx, a, locationID); err != nil {
		return err
	}
	for _, rule := range rules {
		if err := s.Reorder.Save(ctx, rule); err != nil {
			return err
		}
	}
	s.reorderChanges.Add(ctx, int64(len(rules)), metric.WithAttributes(attribute.String("action", "set")))
	return nil
}
