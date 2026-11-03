package app

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// SalesSettingsUpdate holds the sales rules to change; a nil field keeps its
// current value.
type SalesSettingsUpdate struct {
	// MaxDiscount is the discount limit of users without sales:discount, as
	// a fraction of the line's gross amount.
	MaxDiscount        *decimal.Decimal
	RequireCashSession *bool
}

// SalesSettings returns the actor's tenant sales rules: the defaults (no
// discount, cash session required) if it never changed them.
func (s *Service) SalesSettings(ctx context.Context, a Actor) (_ domain.SalesPolicy, err error) {
	ctx, span := s.startSpan(ctx, "GetSalesSettings", a)
	defer func() { usecase.End(span, err) }()

	return s.Policies.Get(ctx, a.TenantID)
}

// UpdateSalesSettings changes the sales rules of the actor's tenant; they
// apply from the next sale. ErrInvalidDiscountLimit unless 0 ≤ MaxDiscount
// ≤ 1 with up to four decimals, and then nothing changes. The permission is
// checked by the API (tenant:manage).
func (s *Service) UpdateSalesSettings(ctx context.Context, a Actor, in SalesSettingsUpdate) (_ domain.SalesPolicy, err error) {
	ctx, span := s.startSpan(ctx, "UpdateSalesSettings", a)
	defer func() { usecase.End(span, err) }()

	policy, err := s.Policies.Get(ctx, a.TenantID)
	if err != nil || in == (SalesSettingsUpdate{}) {
		return policy, err
	}
	var changed []string
	if in.MaxDiscount != nil {
		if policy.MaxDiscount, err = domain.NewDiscountLimit(*in.MaxDiscount); err != nil {
			return domain.SalesPolicy{}, err
		}
		changed = append(changed, "max_discount")
	}
	if in.RequireCashSession != nil {
		policy.CashSessionOptional = !*in.RequireCashSession
		changed = append(changed, "require_cash_session")
	}
	if err := s.Policies.Save(ctx, a.TenantID, policy); err != nil {
		return domain.SalesPolicy{}, err
	}
	for _, name := range changed {
		s.settingsChanges.Add(ctx, 1, metric.WithAttributes(attribute.String("setting", name)))
	}
	s.Logger.InfoContext(ctx, "sales settings updated",
		slog.String("max_discount", policy.MaxDiscount.Max().String()),
		slog.Bool("require_cash_session", !policy.CashSessionOptional))
	return policy, nil
}
