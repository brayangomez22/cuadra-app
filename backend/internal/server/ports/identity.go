package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	catalogapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	catalogdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers"
	identityapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	identitydomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/templates"
	"github.com/brayangomez22/cuadra-app/backend/templates/business"
)

// TenantSetup satisfies identityapp.TenantSetup with the business templates
// embedded in the binary: a new tenant gets its template's catalog (units,
// categories, attributes and variant axes), customer segments and modules.
// Called with the sign-up's context, the catalog's, the customers' and the
// modules' writes join its transaction.
type TenantSetup struct {
	templates *templates.Catalog
	catalog   *catalog.Module
	customers *customers.Module
	modules   *modules.Service
}

// NewTenantSetup loads the embedded templates, checking their modules
// against registry. It fails, so the API does not start, if a template is
// invalid.
func NewTenantSetup(registry *modules.Registry, c *catalog.Module, cu *customers.Module, m *modules.Service) (TenantSetup, error) {
	catalogs, err := templates.Load(business.FS, registry.CheckSet)
	if err != nil {
		return TenantSetup{}, err
	}
	return TenantSetup{templates: catalogs, catalog: c, customers: cu, modules: m}, nil
}

// Templates lists the embedded templates, in the order the sign-up offers them.
func (s TenantSetup) Templates() []identityapp.BusinessTemplate {
	list := s.templates.List()
	out := make([]identityapp.BusinessTemplate, 0, len(list))
	for _, t := range list {
		out = append(out, identityapp.BusinessTemplate{
			ID: t.ID, Name: t.Name, Description: t.Description, DefaultTaxRate: t.DefaultTaxRate,
		})
	}
	return out
}

// SetUpTenant seeds the catalog and the customer segments and turns on the
// modules of template id for the new tenant.
func (s TenantSetup) SetUpTenant(ctx context.Context, tenantID uuid.UUID, id string) error {
	t, ok := s.templates.Get(id)
	if !ok {
		return identitydomain.ErrUnknownBusinessTemplate
	}
	if err := s.catalog.SeedTenant(ctx, tenantID, catalogSeed(t)); err != nil {
		return err
	}
	if err := s.customers.SeedSegments(ctx, tenantID, t.CustomerSegments); err != nil {
		return err
	}
	return s.modules.SetUpTenant(ctx, tenantID, t.Modules)
}

// catalogSeed is the catalog's part of template t: what SeedTenant stores.
func catalogSeed(t templates.Template) catalogapp.Seed {
	seed := catalogapp.Seed{Categories: t.Categories}
	for _, u := range t.Units {
		seed.Units = append(seed.Units, catalogapp.UnitSeed{Code: u.Code, Name: u.Name})
	}
	for _, a := range t.Attributes {
		seed.Attributes = append(seed.Attributes, catalogdomain.AttributeParams{
			Key: a.Key, Label: a.Label, Type: catalogdomain.AttributeType(a.Type), Options: a.Options, Required: a.Required,
		})
	}
	for _, a := range t.VariantAxes {
		seed.VariantAxes = append(seed.VariantAxes, catalogapp.AxisSeed{Name: a.Name, Values: a.Values})
	}
	return seed
}

// IdentityRegistry satisfies identityapp.ModuleRegistry: the permissions
// each registered module declares.
type IdentityRegistry struct{ Registry *modules.Registry }

// Permissions lists the permissions each registered module declares.
func (r IdentityRegistry) Permissions() []identityapp.ModulePermission {
	var out []identityapp.ModulePermission
	for _, d := range r.Registry.Definitions() {
		for _, p := range d.Permissions {
			out = append(out, identityapp.ModulePermission{Module: d.ID, Permission: p.ID, Roles: p.Roles})
		}
	}
	return out
}
