package main

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	catalogapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	catalogdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	identityapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	identitydomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/templates"
	"github.com/brayangomez22/cuadra-app/backend/templates/business"
)

// tenantSetup satisfies identityapp.TenantSetup with the business templates
// embedded in the binary: a new tenant gets its template's catalog (units,
// categories, attributes and variant axes) and modules. Called with the
// sign-up's context, the catalog's and the modules' writes join its
// transaction.
type tenantSetup struct {
	templates *templates.Catalog
	catalog   *catalog.Module
	modules   *modules.Service
}

// newTenantSetup loads the embedded templates, checking their modules
// against registry. It fails, so the API does not start, if a template is
// invalid.
func newTenantSetup(registry *modules.Registry, c *catalog.Module, m *modules.Service) (tenantSetup, error) {
	catalogs, err := templates.Load(business.FS, registry.CheckSet)
	if err != nil {
		return tenantSetup{}, err
	}
	return tenantSetup{templates: catalogs, catalog: c, modules: m}, nil
}

func (s tenantSetup) Templates() []identityapp.BusinessTemplate {
	list := s.templates.List()
	out := make([]identityapp.BusinessTemplate, 0, len(list))
	for _, t := range list {
		out = append(out, identityapp.BusinessTemplate{ID: t.ID, Name: t.Name, Description: t.Description})
	}
	return out
}

func (s tenantSetup) SetUpTenant(ctx context.Context, tenantID uuid.UUID, id string) error {
	t, ok := s.templates.Get(id)
	if !ok {
		return identitydomain.ErrUnknownBusinessTemplate
	}
	seed := catalogapp.Seed{Categories: t.Categories}
	for _, u := range t.Units {
		seed.Units = append(seed.Units, catalogapp.UnitSeed{Code: u.Code, Name: u.Name})
	}
	for _, a := range t.Attributes {
		seed.Attributes = append(seed.Attributes, catalogdomain.AttributeParams{
			Key: a.Key, Label: a.Label, Type: catalogdomain.AttributeType(a.Type), Options: a.Options,
		})
	}
	for _, a := range t.VariantAxes {
		seed.VariantAxes = append(seed.VariantAxes, catalogapp.AxisSeed{Name: a.Name, Values: a.Values})
	}
	if err := s.catalog.SeedTenant(ctx, tenantID, seed); err != nil {
		return err
	}
	return s.modules.SetUpTenant(ctx, tenantID, t.Modules)
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
