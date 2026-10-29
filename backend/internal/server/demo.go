package server

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
)

// DemoTenant describes a demonstration tenant and its staff.
type DemoTenant = identity.DemoTenantInput

// DemoUser is a member of a demo tenant's staff.
type DemoUser = identity.DemoUser

// The roles of a demo tenant's staff.
const (
	DemoRoleAdmin     = identity.RoleAdmin
	DemoRoleCashier   = identity.RoleCashier
	DemoRoleWarehouse = identity.RoleWarehouse
)

// Demo does for the demo generator (internal/demo) what the contract does
// not offer: signing up a tenant marked demo with one user per role, and
// deleting it. Everything else the generator writes goes through the API.
type Demo struct{ identity *identity.Module }

// NewDemo builds the API as New does, and its Demo.
func NewDemo(cfg Config) (http.Handler, *Demo, error) {
	h, w, err := build(cfg)
	if err != nil {
		return nil, nil, err
	}
	return h, &Demo{identity: w.identityModule}, nil
}

// SetUpTenant signs up a demo tenant with its template's data and staff.
func (d *Demo) SetUpTenant(ctx context.Context, in DemoTenant) (uuid.UUID, error) {
	return d.identity.SetUpDemoTenant(ctx, in)
}

// Tenants returns the ids of the demo tenants of a business template.
func (d *Demo) Tenants(ctx context.Context, template string) ([]uuid.UUID, error) {
	return d.identity.DemoTenants(ctx, template)
}

// Purge deletes a demo tenant with all its data; it refuses any other.
func (d *Demo) Purge(ctx context.Context, id uuid.UUID) error {
	return d.identity.PurgeDemoTenant(ctx, id)
}
