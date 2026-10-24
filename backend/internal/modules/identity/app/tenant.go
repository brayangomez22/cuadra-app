package app

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// TenantUpdate holds the tenant settings to change; a nil field keeps its
// current value.
type TenantUpdate struct {
	DefaultTaxRate *decimal.Decimal
}

// UpdateTenant changes the settings of userID's tenant. The permission is
// checked by the API (tenant:manage); an invalid value changes nothing.
func (s *Service) UpdateTenant(ctx context.Context, tenantID, userID uuid.UUID, in TenantUpdate) (_ *domain.Tenant, err error) {
	ctx, span := s.startSpan(ctx, "UpdateTenant")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(attribute.String("tenant.id", tenantID.String()), attribute.String("user.id", userID.String()))

	tenant, err := s.Tenants.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if in.DefaultTaxRate == nil {
		return tenant, nil
	}
	if err := tenant.SetDefaultTaxRate(*in.DefaultTaxRate); err != nil {
		return nil, err
	}
	if err := s.Tenants.Update(ctx, tenant); err != nil {
		return nil, err
	}
	s.Logger.InfoContext(ctx, "tenant settings updated", slog.String("default_tax_rate", tenant.DefaultTaxRate().String()))
	return tenant, nil
}
