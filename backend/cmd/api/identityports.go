package main

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	identityapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
)

// tenantSetup satisfies identityapp.TenantSetup: a new tenant starts with
// the catalog's default units of measure. Called with the sign-up's context,
// the catalog's writes join its transaction.
type tenantSetup struct{ catalog *catalog.Module }

func (s tenantSetup) SetUpTenant(ctx context.Context, tenantID uuid.UUID) error {
	return s.catalog.SeedDefaultUnits(ctx, tenantID)
}

// identityRegistry satisfies identityapp.ModuleRegistry: the permissions
// each registered module declares.
type identityRegistry struct{ registry *modules.Registry }

func (r identityRegistry) Permissions() []identityapp.ModulePermission {
	var out []identityapp.ModulePermission
	for _, d := range r.registry.Definitions() {
		for _, p := range d.Permissions {
			out = append(out, identityapp.ModulePermission{Module: d.ID, Permission: p.ID, Roles: p.Roles})
		}
	}
	return out
}
