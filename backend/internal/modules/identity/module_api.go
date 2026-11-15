package identity

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
)

// DemoTenantInput describes a demonstration tenant (see
// app.Service.SetUpDemoTenant).
type DemoTenantInput = app.DemoTenantInput

// DemoUser is a member of a demo tenant's staff.
type DemoUser = app.DemoUser

// The roles a demo tenant's staff can have.
const (
	RoleAdmin     = "admin"
	RoleCashier   = "cashier"
	RoleWarehouse = "warehouse"
)

// SetUpDemoTenant signs up a demo tenant with its staff. It is for the demo
// generator, through internal/server.
func (m *Module) SetUpDemoTenant(ctx context.Context, in DemoTenantInput) (uuid.UUID, error) {
	return m.service.SetUpDemoTenant(ctx, in)
}

// DemoTenants returns the ids of the demo tenants of a business template.
func (m *Module) DemoTenants(ctx context.Context, template string) ([]uuid.UUID, error) {
	return m.service.DemoTenants(ctx, template)
}

// PurgeDemoTenant deletes a demo tenant with all its data; it refuses any
// other tenant.
func (m *Module) PurgeDemoTenant(ctx context.Context, id uuid.UUID) error {
	return m.service.PurgeDemoTenant(ctx, id)
}

// TenantProfile is what other modules read of a tenant: the company that
// issues the electronic documents.
type TenantProfile struct {
	Name             string
	NIT              string
	Address          string
	City             string
	MunicipalityCode string
	Phone            string
	Demo             bool
}

// TenantProfile returns the tenant's company.
func (m *Module) TenantProfile(ctx context.Context, tenantID uuid.UUID) (TenantProfile, error) {
	t, err := m.service.Tenant(ctx, tenantID)
	if err != nil {
		return TenantProfile{}, err
	}
	c := t.Contact()
	return TenantProfile{
		Name: t.Name(), NIT: t.NIT().String(), Address: c.Address(), City: c.City(),
		MunicipalityCode: c.MunicipalityCode(), Phone: c.Phone(), Demo: t.IsDemo(),
	}, nil
}
