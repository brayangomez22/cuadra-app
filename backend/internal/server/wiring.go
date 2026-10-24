package server

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	"github.com/brayangomez22/cuadra-app/backend/internal/server/ports"
)

// wiring builds the business modules on what they share: the API's
// configuration, the activable modules and the permission matrix.
type wiring struct {
	Config
	registry      *modules.Registry
	moduleService *modules.Service
	permissions   *identity.Permissions
}

// newWiring builds the activable modules and the permission matrix, which
// the business modules authorize with.
func newWiring(cfg Config) (*wiring, error) {
	registry, err := modules.NewRegistry(modules.Builtin()...)
	if err != nil {
		return nil, err
	}
	moduleService, err := newModuleService(registry, cfg)
	if err != nil {
		return nil, err
	}
	permissions, err := identity.NewPermissions(ports.IdentityRegistry{Registry: registry})
	if err != nil {
		return nil, err
	}
	return &wiring{Config: cfg, registry: registry, moduleService: moduleService, permissions: permissions}, nil
}

// modules builds every business module, joined through the ports of each
// one, and returns their routes.
func (w *wiring) modules() ([]func(*http.ServeMux), error) {
	catalogModule, err := w.catalog()
	if err != nil {
		return nil, err
	}
	identityModule, err := w.identity(catalogModule)
	if err != nil {
		return nil, err
	}
	inventoryModule, err := w.inventory(catalogModule)
	if err != nil {
		return nil, err
	}
	salesModule, err := w.sales(catalogModule, inventoryModule)
	if err != nil {
		return nil, err
	}
	return []func(*http.ServeMux){
		identityModule.RegisterRoutes, catalogModule.RegisterRoutes, inventoryModule.RegisterRoutes,
		salesModule.RegisterRoutes, moduleRoutes(w.moduleService, w.Log),
	}, nil
}

func (w *wiring) catalog() (*catalog.Module, error) {
	return catalog.New(catalog.Config{
		DB:             w.DB,
		Authorize:      w.permissions.Authorize,
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
	})
}

// identity signs tenants up with the catalog and the modules of their
// business template.
func (w *wiring) identity(c *catalog.Module) (*identity.Module, error) {
	setup, err := ports.NewTenantSetup(w.registry, c, w.moduleService)
	if err != nil {
		return nil, err
	}
	return identity.New(identity.Config{
		DB:             w.DB,
		Issuer:         w.Tokens,
		Setup:          setup,
		Permissions:    w.permissions,
		Modules:        w.moduleService,
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
	})
}

func (w *wiring) inventory(c *catalog.Module) (*inventory.Module, error) {
	return inventory.New(inventory.Config{
		DB:             w.DB,
		Authorize:      w.permissions.Authorize,
		Variants:       ports.InventoryCatalog{Catalog: c},
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
	})
}

func (w *wiring) sales(c *catalog.Module, i *inventory.Module) (*sales.Module, error) {
	return sales.New(sales.Config{
		DB:             w.DB,
		Authorize:      w.permissions.Authorize,
		Catalog:        ports.SalesCatalog{Catalog: c},
		Stock:          ports.SalesStock{Inventory: i},
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
	})
}
