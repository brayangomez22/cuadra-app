package app

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// GetSettings returns the tenant's invoicing settings.
func (s *Service) GetSettings(ctx context.Context, a Actor) (_ domain.Settings, err error) {
	ctx, span := s.startSpan(ctx, "GetSettings", a)
	defer func() { usecase.End(span, err) }()

	return s.Settings.Get(ctx, a.TenantID)
}

// UpdateSettings turns emission on or off and sets the default kind.
func (s *Service) UpdateSettings(ctx context.Context, a Actor, enabled bool, kind string) (_ domain.Settings, err error) {
	ctx, span := s.startSpan(ctx, "UpdateSettings", a, attribute.Bool("invoicing.enabled", enabled))
	defer func() { usecase.End(span, err) }()

	settings, err := domain.NewSettings(enabled, domain.Kind(kind))
	if err != nil {
		return domain.Settings{}, err
	}
	if err := s.Settings.Save(ctx, a.TenantID, settings); err != nil {
		return domain.Settings{}, err
	}
	s.Logger.InfoContext(ctx, "invoicing settings changed", slog.Bool("enabled", enabled), slog.String("default_kind", kind))
	return settings, nil
}
