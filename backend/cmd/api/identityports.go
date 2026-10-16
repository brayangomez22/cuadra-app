package main

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
)

// tenantSetup satisfies identityapp.TenantSetup: a new tenant starts with
// the catalog's default units of measure. Called with the sign-up's context,
// the catalog's writes join its transaction.
type tenantSetup struct{ catalog *catalog.Module }

func (s tenantSetup) SetUpTenant(ctx context.Context, tenantID uuid.UUID) error {
	return s.catalog.SeedDefaultUnits(ctx, tenantID)
}
