package server

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables"
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
	// identityModule is kept for the demo (NewDemo).
	identityModule *identity.Module
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
	// The catalog asks the inventory whether a product has stock movements,
	// and the inventory reads the catalog's variants: the history is joined
	// once the inventory exists.
	history := &ports.CatalogHistory{}
	catalogModule, err := w.catalog(history)
	if err != nil {
		return nil, err
	}
	customersModule, err := w.customers()
	if err != nil {
		return nil, err
	}
	identityModule, err := w.identity(catalogModule, customersModule)
	if err != nil {
		return nil, err
	}
	w.identityModule = identityModule
	inventoryModule, err := w.inventory(catalogModule)
	if err != nil {
		return nil, err
	}
	history.Inventory = inventoryModule
	// Sales charge credit sales to the receivables, whose cash payments go
	// into the sales module's drawers: the credit is joined once both exist.
	credit := &ports.SalesCredit{}
	salesModule, err := w.sales(catalogModule, inventoryModule, customersModule, salesCredit{credit, identityModule})
	if err != nil {
		return nil, err
	}
	receivablesModule, err := w.receivables(customersModule, salesModule)
	if err != nil {
		return nil, err
	}
	credit.Receivables = receivablesModule
	importsModule, err := w.imports(catalogModule, identityModule, inventoryModule)
	if err != nil {
		return nil, err
	}
	purchasesModule, err := w.purchases(catalogModule, inventoryModule)
	if err != nil {
		return nil, err
	}
	return []func(*http.ServeMux){
		identityModule.RegisterRoutes, catalogModule.RegisterRoutes, customersModule.RegisterRoutes,
		inventoryModule.RegisterRoutes, salesModule.RegisterRoutes, receivablesModule.RegisterRoutes,
		importsModule.RegisterRoutes, purchasesModule.RegisterRoutes, moduleRoutes(w.moduleService, w.Log),
	}, nil
}

func (w *wiring) catalog(history *ports.CatalogHistory) (*catalog.Module, error) {
	return catalog.New(catalog.Config{
		DB:             w.DB,
		Authorize:      w.permissions.Authorize,
		History:        history,
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}

func (w *wiring) customers() (*customers.Module, error) {
	return customers.New(customers.Config{
		DB:             w.DB,
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}

// identity signs tenants up with the catalog, the customer segments and the
// modules of their business template.
func (w *wiring) identity(c *catalog.Module, cu *customers.Module) (*identity.Module, error) {
	setup, err := ports.NewTenantSetup(w.registry, c, cu, w.moduleService)
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
		Now:            w.Now,
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
		Now:            w.Now,
	})
}

// salesCredit is how sales on credit reach the receivables (set once they
// exist) and who authorizes one over the customer's credit.
type salesCredit struct {
	ledger   *ports.SalesCredit
	identity *identity.Module
}

// sales sells the catalog's variants from the inventory's stock, to the
// customers module's customers, on credit through the receivables.
func (w *wiring) sales(c *catalog.Module, i *inventory.Module, cu *customers.Module, cr salesCredit) (*sales.Module, error) {
	return sales.New(sales.Config{
		DB:             w.DB,
		Authorize:      w.permissions.Authorize,
		Catalog:        ports.SalesCatalog{Catalog: c},
		Stock:          ports.SalesStock{Inventory: i},
		Customers:      ports.SalesCustomers{Customers: cu},
		Credit:         cr.ledger,
		Authorizer:     ports.SalesAuthorizer{Identity: cr.identity},
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}

// receivables gives credit to the customers module's customers and takes
// their cash payments into the sales module's drawers.
func (w *wiring) receivables(cu *customers.Module, s *sales.Module) (*receivables.Module, error) {
	return receivables.New(receivables.Config{
		DB:             w.DB,
		Customers:      ports.ReceivablesCustomers{Customers: cu},
		Cash:           ports.ReceivablesCash{Sales: s},
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}

// imports loads files into the catalog (with the tenant's default IVA) and
// the stock, and works its job in the process's workers.
func (w *wiring) imports(c *catalog.Module, id *identity.Module, i *inventory.Module) (*imports.Module, error) {
	return imports.New(imports.Config{
		DB:             w.DB,
		Jobs:           w.Jobs,
		Workers:        w.Workers,
		Schedule:       w.Schedule,
		Catalog:        ports.ImportsCatalog{Catalog: c, Identity: id},
		Stock:          ports.ImportsStock{Inventory: i},
		Modules:        w.moduleService,
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}

// purchases buys the catalog's variants into the inventory's stock.
func (w *wiring) purchases(c *catalog.Module, i *inventory.Module) (*purchases.Module, error) {
	return purchases.New(purchases.Config{
		DB:             w.DB,
		Catalog:        ports.PurchasesCatalog{Catalog: c},
		Stock:          ports.PurchasesStock{Inventory: i},
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}
