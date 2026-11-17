package server

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/analytics"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	"github.com/brayangomez22/cuadra-app/backend/internal/server/ports"
)

// wiring builds the business modules on what they share: the API's
// configuration, the activable modules and the permission matrix.
type wiring struct {
	Config
	registry      *modules.Registry
	moduleService *modules.Service
	flagService   *flags.Service
	permissions   *identity.Permissions
	// identityModule is kept for the demo (NewDemo).
	identityModule *identity.Module
	// auditor writes to the audit log for every audited module.
	auditor *ports.Auditor
}

// newWiring builds the audit log, the activable modules (whose changes it
// records) and the permission matrix, which the business modules authorize
// with.
func newWiring(cfg Config) (*wiring, error) {
	registry, err := modules.NewRegistry(modules.Builtin()...)
	if err != nil {
		return nil, err
	}
	w := &wiring{Config: cfg, registry: registry}
	auditModule, err := w.auditLog()
	if err != nil {
		return nil, err
	}
	w.auditor = &ports.Auditor{Audit: auditModule}
	if w.moduleService, err = newModuleService(registry, cfg, w.auditor); err != nil {
		return nil, err
	}
	if w.flagService, err = newFlagService(cfg); err != nil {
		return nil, err
	}
	permissions, err := identity.NewPermissions(ports.IdentityRegistry{Registry: registry})
	if err != nil {
		return nil, err
	}
	w.permissions = permissions
	return w, nil
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
	w.auditor.Identity = identityModule
	inventoryModule, err := w.inventory(catalogModule)
	if err != nil {
		return nil, err
	}
	history.Inventory = inventoryModule
	sell, err := w.selling(catalogModule, inventoryModule, customersModule, identityModule)
	if err != nil {
		return nil, err
	}
	importsModule, err := w.imports(catalogModule, identityModule, inventoryModule)
	if err != nil {
		return nil, err
	}
	purchasesModule, err := w.purchases(catalogModule, inventoryModule)
	if err != nil {
		return nil, err
	}
	feedbackModule, err := w.feedback()
	if err != nil {
		return nil, err
	}
	return []func(*http.ServeMux){
		identityModule.RegisterRoutes, catalogModule.RegisterRoutes, customersModule.RegisterRoutes,
		inventoryModule.RegisterRoutes, sell.sales.RegisterRoutes, sell.receivables.RegisterRoutes,
		sell.invoicing.RegisterRoutes, importsModule.RegisterRoutes, purchasesModule.RegisterRoutes, moduleRoutes(w.moduleService, w.Log),
		w.auditor.Audit.RegisterRoutes, flagRoutes(w.flagService, w.Log), feedbackModule.RegisterRoutes,
	}, nil
}

func (w *wiring) catalog(history *ports.CatalogHistory) (*catalog.Module, error) {
	return catalog.New(catalog.Config{
		DB:             w.DB,
		Authorize:      w.permissions.Authorize,
		History:        history,
		Audit:          ports.CatalogAudit{Auditor: w.auditor},
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}

func (w *wiring) auditLog() (*audit.Module, error) {
	return audit.New(audit.Config{
		DB:             w.DB,
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}

// signupLimiter is the sign-ups' limit of the API's RateLimits, or none.
func (w *wiring) signupLimiter() identity.RateLimiter {
	if l := w.RateLimits.signupLimiter(); l != nil {
		return l
	}
	return nil
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
		Flags:          w.flagService,
		Pseudonyms:     analytics.NewPseudonyms(w.AnalyticsSecret),
		Audit:          ports.IdentityAudit{Auditor: w.auditor},
		SignupLimiter:  w.signupLimiter(),
		Jobs:           w.Jobs,
		Workers:        w.Workers,
		Schedule:       w.Schedule,
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
		DemoEnabled:    w.DemoEnabled,
	})
}

func (w *wiring) inventory(c *catalog.Module) (*inventory.Module, error) {
	return inventory.New(inventory.Config{
		DB:             w.DB,
		Authorize:      w.permissions.Authorize,
		Variants:       ports.InventoryCatalog{Catalog: c},
		Audit:          ports.InventoryAudit{Auditor: w.auditor},
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
		Audit:          ports.PurchasesAudit{Auditor: w.auditor},
		Logger:         w.Log,
		TracerProvider: w.TracerProvider,
		MeterProvider:  w.MeterProvider,
		Now:            w.Now,
	})
}
