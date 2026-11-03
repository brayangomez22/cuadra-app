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
// current value and an empty text clears it.
type TenantUpdate struct {
	DefaultTaxRate *decimal.Decimal
	Address        *string
	City           *string
	Phone          *string
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
	if in == (TenantUpdate{}) {
		return tenant, nil
	}
	contact, err := in.contact(tenant.Contact())
	if err != nil {
		return nil, err
	}
	if in.DefaultTaxRate != nil {
		if err := tenant.SetDefaultTaxRate(*in.DefaultTaxRate); err != nil {
			return nil, err
		}
	}
	tenant.SetContact(contact)
	if err := s.Tenants.Update(ctx, tenant); err != nil {
		return nil, err
	}
	s.Logger.InfoContext(ctx, "tenant settings updated", slog.String("default_tax_rate", tenant.DefaultTaxRate().String()))
	return tenant, nil
}

// contact returns current with the fields of the update that are not nil.
func (in TenantUpdate) contact(current domain.Contact) (domain.Contact, error) {
	address, city, phone := current.Address(), current.City(), current.Phone()
	if in.Address != nil {
		address = *in.Address
	}
	if in.City != nil {
		city = *in.City
	}
	if in.Phone != nil {
		phone = *in.Phone
	}
	return domain.NewContact(address, city, phone)
}

// DefaultTaxRate returns the IVA the tenant proposes to a new product. It
// is for other modules: an import gives it to the rows without IVA.
func (s *Service) DefaultTaxRate(ctx context.Context, tenantID uuid.UUID) (_ decimal.Decimal, err error) {
	ctx, span := s.startSpan(ctx, "GetDefaultTaxRate")
	defer func() { usecase.End(span, err) }()
	span.SetAttributes(attribute.String("tenant.id", tenantID.String()))

	tenant, err := s.Tenants.GetByID(ctx, tenantID)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return tenant.DefaultTaxRate(), nil
}
