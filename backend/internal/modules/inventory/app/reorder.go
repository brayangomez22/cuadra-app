package app

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// errReorderMismatch reports a repository alert the domain does not see as
// one: the SQL condition and ReorderRule.NeedsReorder disagree.
var errReorderMismatch = errors.New("inventory: reorder alert the rule does not need")

// ReorderInput is a variant's minimum and maximum stock at a location, in the
// product's base unit.
type ReorderInput struct {
	VariantID  uuid.UUID
	LocationID uuid.UUID
	Min        decimal.Decimal
	Max        decimal.Decimal
}

// ReorderQuery selects the alerts of one location, or of every location
// when LocationID is nil.
type ReorderQuery struct {
	LocationID *uuid.UUID
	Limit      int
	Offset     int
}

// Alert is a reorder alert with the product of its variant.
type Alert struct {
	domain.ReorderAlert
	ProductID uuid.UUID
}

// AlertPage is one page of alerts and how many match in total.
type AlertPage struct {
	Items []Alert
	Total int
}

// SetReorderRule stores the minimum and maximum of a variant at a location,
// replacing the previous ones. ErrInvalidReorderRange unless 0 ≤ min ≤ max;
// ErrLocationNotFound or ErrVariantNotFound if either is not of the actor's
// tenant.
func (s *Service) SetReorderRule(ctx context.Context, a Actor, in ReorderInput) (_ *domain.ReorderRule, err error) {
	ctx, span := s.startSpan(ctx, "SetReorderRule", a, variantAttr(in.VariantID), locationAttr(in.LocationID))
	defer func() { usecase.End(span, err) }()

	rule, err := domain.NewReorderRule(domain.ReorderRuleSpec{
		TenantID: a.TenantID, VariantID: in.VariantID, LocationID: in.LocationID, Min: in.Min, Max: in.Max,
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.Locations.GetByID(ctx, a.TenantID, in.LocationID); err != nil {
		return nil, err
	}
	if err := s.Reorder.Save(ctx, rule); err != nil {
		return nil, err
	}
	s.reorderChanges.Add(ctx, 1, metric.WithAttributes(attribute.String("action", "set")))
	return rule, nil
}

// DeleteReorderRule removes the rule of a variant at a location, so it no
// longer raises alerts; ErrReorderRuleNotFound if there is none.
func (s *Service) DeleteReorderRule(ctx context.Context, a Actor, variantID, locationID uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "DeleteReorderRule", a, variantAttr(variantID), locationAttr(locationID))
	defer func() { usecase.End(span, err) }()

	if err := s.Reorder.Delete(ctx, a.TenantID, variantID, locationID); err != nil {
		return err
	}
	s.reorderChanges.Add(ctx, 1, metric.WithAttributes(attribute.String("action", "delete")))
	return nil
}

// ListReorderAlerts returns one page of the variants at or below their
// minimum, with how much to order, the furthest below it first;
// ErrLocationNotFound if the location given is not of the actor's tenant.
func (s *Service) ListReorderAlerts(ctx context.Context, a Actor, q ReorderQuery) (_ AlertPage, err error) {
	var attrs []attribute.KeyValue
	if q.LocationID != nil {
		attrs = append(attrs, locationAttr(*q.LocationID))
	}
	ctx, span := s.startSpan(ctx, "ListReorderAlerts", a, attrs...)
	defer func() { usecase.End(span, err) }()

	if q.LocationID != nil {
		if _, err := s.Locations.GetByID(ctx, a.TenantID, *q.LocationID); err != nil {
			return AlertPage{}, err
		}
	}
	f := domain.ReorderFilter{LocationID: q.LocationID}
	f.Limit, f.Offset = pageBounds(q.Limit, q.Offset)
	page, err := s.Reorder.Alerts(ctx, a.TenantID, f)
	if err != nil {
		return AlertPage{}, err
	}
	items, err := s.alertsWithProducts(ctx, a, page.Items)
	if err != nil {
		return AlertPage{}, err
	}
	return AlertPage{Items: items, Total: page.Total}, nil
}

// alertsWithProducts turns each rule and its stock into the domain's alert,
// with the product of its variant.
func (s *Service) alertsWithProducts(ctx context.Context, a Actor, rows []domain.RuleStock) ([]Alert, error) {
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.Rule.VariantID())
	}
	products, err := s.ProductsOf(ctx, a, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Alert, 0, len(rows))
	for _, r := range rows {
		alert, ok := r.Rule.Alert(r.OnHand)
		if !ok {
			return nil, errReorderMismatch
		}
		out = append(out, Alert{ReorderAlert: alert, ProductID: products[r.Rule.VariantID()]})
	}
	return out, nil
}

// withReorder adds to each level of a location its reorder rule.
func (s *Service) withReorder(ctx context.Context, a Actor, locationID uuid.UUID, levels []Level) error {
	ids := make([]uuid.UUID, 0, len(levels))
	for _, l := range levels {
		ids = append(ids, l.VariantID())
	}
	rules, err := s.Reorder.ForVariants(ctx, a.TenantID, locationID, ids)
	if err != nil {
		return err
	}
	for i := range levels {
		levels[i].Reorder = rules[levels[i].VariantID()]
	}
	return nil
}

// NeedsReorder reports whether the level is at or below its rule's minimum;
// false without a rule.
func (l Level) NeedsReorder() bool {
	return l.Reorder != nil && l.Reorder.NeedsReorder(l.Quantity())
}
