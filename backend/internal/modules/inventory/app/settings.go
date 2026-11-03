package app

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// StockSettingsUpdate holds the stock rules to change; a nil field keeps its
// current value.
type StockSettingsUpdate struct {
	AllowNegative *bool
}

// StockSettings returns the actor's tenant stock rules: the defaults if it
// never changed them.
func (s *Service) StockSettings(ctx context.Context, a Actor) (_ domain.StockPolicy, err error) {
	ctx, span := s.startSpan(ctx, "GetStockSettings", a)
	defer func() { usecase.End(span, err) }()

	return s.Policies.Get(ctx, a.TenantID)
}

// UpdateStockSettings changes the stock rules of the actor's tenant; they
// apply from the next movement. The permission is checked by the API
// (tenant:manage).
func (s *Service) UpdateStockSettings(ctx context.Context, a Actor, in StockSettingsUpdate) (_ domain.StockPolicy, err error) {
	ctx, span := s.startSpan(ctx, "UpdateStockSettings", a)
	defer func() { usecase.End(span, err) }()

	policy, err := s.Policies.Get(ctx, a.TenantID)
	if err != nil || in.AllowNegative == nil {
		return policy, err
	}
	policy.AllowNegative = *in.AllowNegative
	if err := s.Policies.Save(ctx, a.TenantID, policy); err != nil {
		return domain.StockPolicy{}, err
	}
	s.settingsChanges.Add(ctx, 1, metric.WithAttributes(attribute.String("setting", "allow_negative_stock")))
	s.Logger.InfoContext(ctx, "stock settings updated", slog.Bool("allow_negative_stock", policy.AllowNegative))
	return policy, nil
}
