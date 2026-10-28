package ports

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	catalogapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
	importsapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	importsdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory"
	inventoryapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	inventorydomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// The imports module reaches the catalog (with the tenant's default IVA,
// from identity) and the inventory through its own ports
// (importsapp.Catalog, importsapp.Stock); these adapters translate the
// rows of a file into the catalog's drafts and the catalog's errors into
// row problems.

// ImportsCatalog satisfies importsapp.Catalog.
type ImportsCatalog struct {
	Catalog  *catalog.Module
	Identity *identity.Module
}

// Layout returns the tenant's columns: its units, attributes and variant
// axes, and its default IVA.
func (c ImportsCatalog) Layout(ctx context.Context, a importsapp.Actor) (importsdomain.Layout, error) {
	cols, err := c.Catalog.ImportColumns(ctx, a.TenantID, a.UserID)
	if err != nil {
		return importsdomain.Layout{}, err
	}
	rate, err := c.Identity.DefaultTaxRate(ctx, a.TenantID)
	if err != nil {
		return importsdomain.Layout{}, err
	}
	l := importsdomain.Layout{DefaultTaxRate: rate}
	for _, u := range cols.Units {
		l.Units = append(l.Units, u.Code())
	}
	for _, d := range cols.Attributes {
		l.Attributes = append(l.Attributes, importsdomain.AttributeColumn{
			Key: d.Key(), Label: d.Label(), Type: importsdomain.AttributeKind(d.Type()),
			Required: d.Required(), Options: d.Options(),
		})
	}
	for _, axis := range cols.Axes {
		l.Axes = append(l.Axes, importsdomain.AxisColumn{Name: axis.Name(), Values: axis.Values()})
	}
	return l, nil
}

// CheckProducts asks the catalog what writing each group would do.
func (c ImportsCatalog) CheckProducts(ctx context.Context, a importsapp.Actor, groups []importsdomain.Group, p importsdomain.ExistingPolicy) ([]importsapp.ProductResult, error) {
	results, err := c.Catalog.CheckProductDrafts(ctx, a.TenantID, a.UserID, drafts(groups), catalogapp.ExistingPolicy(p))
	if err != nil {
		return nil, err
	}
	return productResults(results)
}

// ImportProducts writes the groups with the catalog.
func (c ImportsCatalog) ImportProducts(ctx context.Context, a importsapp.Actor, groups []importsdomain.Group, p importsdomain.ExistingPolicy) ([]importsapp.ProductResult, error) {
	results, err := c.Catalog.ImportProductDrafts(ctx, a.TenantID, a.UserID, drafts(groups), catalogapp.ExistingPolicy(p))
	if err != nil {
		return nil, err
	}
	return productResults(results)
}

// ImportsStock satisfies importsapp.Stock with the inventory module.
type ImportsStock struct{ Inventory *inventory.Module }

// RequireLocation checks the import's location.
func (s ImportsStock) RequireLocation(ctx context.Context, a importsapp.Actor, id uuid.UUID) error {
	err := s.Inventory.CheckLocation(ctx, a.TenantID, a.UserID, id)
	if errors.Is(err, inventorydomain.ErrLocationNotFound) || errors.Is(err, inventorydomain.ErrLocationInactive) {
		return importsapp.ErrLocationNotFound
	}
	return err
}

// ReceiveInitial registers the initial stock, in the batch's transaction.
// A location deactivated after the upload fails the batch like a missing
// one.
func (s ImportsStock) ReceiveInitial(ctx context.Context, a importsapp.Actor, locationID uuid.UUID, entries []importsapp.StockEntry) error {
	lines := make([]inventoryapp.InitialStockLine, 0, len(entries))
	for _, e := range entries {
		lines = append(lines, inventoryapp.InitialStockLine{VariantID: e.VariantID, Quantity: e.Quantity, UnitCost: e.UnitCost})
	}
	err := s.Inventory.ReceiveInitialStock(ctx, a.TenantID, a.UserID, locationID, lines)
	if errors.Is(err, inventorydomain.ErrLocationNotFound) || errors.Is(err, inventorydomain.ErrLocationInactive) {
		return importsapp.ErrLocationNotFound
	}
	return err
}

// CatalogHistory satisfies catalogapp.StockHistory with the inventory
// module. The inventory is built after the catalog (it reads the catalog's
// variants), so the wiring sets Inventory once it exists.
type CatalogHistory struct{ Inventory *inventory.Module }

// HasMovements reports whether any of variantIDs has a stock movement.
func (h *CatalogHistory) HasMovements(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) (bool, error) {
	if h.Inventory == nil {
		return false, errors.New("ports: the inventory is not wired to the catalog")
	}
	return h.Inventory.HasMovements(ctx, tenantID, variantIDs)
}
