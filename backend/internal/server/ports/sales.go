package ports

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory"
	inventoryapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	inventorydomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	salesapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	salesdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// The sales module reaches the catalog and the inventory through its own
// ports (salesapp.VariantCatalog, salesapp.StockReserver), and the inventory
// reaches the catalog through inventoryapp.ProductVariants; these adapters
// satisfy them with the other modules and translate their errors, so no
// module imports another.

// SalesCatalog satisfies salesapp.VariantCatalog with the catalog module.
type SalesCatalog struct{ Catalog *catalog.Module }

// Variants returns what a sale needs of each variant of tenantID among ids.
func (c SalesCatalog) Variants(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]salesapp.VariantInfo, error) {
	products, err := c.Catalog.ProductsOfVariants(ctx, tenantID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]salesapp.VariantInfo, 0, len(ids))
	for _, p := range products {
		for _, v := range p.Variants() {
			if !slices.Contains(ids, v.ID()) {
				continue
			}
			info := salesapp.VariantInfo{
				ID: v.ID(), ProductID: p.ID(), ProductName: p.Name(), Label: v.Label(),
				Price: v.Price(), TaxRate: p.TaxRate().Rate(), Active: p.IsActive() && v.IsActive(),
			}
			for _, u := range v.Units() {
				info.Units = append(info.Units, salesapp.UnitInfo{
					UnitID: u.Unit().ID, Code: u.Unit().Code, Factor: u.Factor(), Price: u.Price(),
				})
			}
			out = append(out, info)
		}
	}
	return out, nil
}

// SalesStock satisfies salesapp.StockReserver with the inventory module: a
// sale takes its stock out of the variants it sold.
type SalesStock struct{ Inventory *inventory.Module }

// ReserveSale takes the stock of the sale's lines, in its transaction.
func (s SalesStock) ReserveSale(ctx context.Context, tenantID, userID, locationID, saleID uuid.UUID, lines []salesapp.StockLine) error {
	in := inventoryapp.SaleStockInput{LocationID: locationID, SaleID: saleID}
	for _, l := range lines {
		in.Lines = append(in.Lines, inventoryapp.SaleStockLine{VariantID: l.VariantID, Quantity: l.Quantity})
	}
	return salesStockError(s.Inventory.IssueSaleStock(ctx, tenantID, userID, in))
}

// ReverseSale gives back the stock the sale took.
func (s SalesStock) ReverseSale(ctx context.Context, tenantID, userID, saleID uuid.UUID) error {
	return salesStockError(s.Inventory.ReverseSaleStock(ctx, tenantID, userID, saleID))
}

// salesStockError translates the inventory's errors into the sales domain
// errors the sales module maps to HTTP. Others pass through as unexpected.
func salesStockError(err error) error {
	var variantErr *inventoryapp.VariantStockError
	switch {
	case err == nil:
		return nil
	case errors.Is(err, inventorydomain.ErrInsufficientStock) && errors.As(err, &variantErr):
		return &salesdomain.InsufficientStockError{VariantID: variantErr.VariantID}
	case errors.Is(err, inventorydomain.ErrInsufficientStock):
		return salesdomain.ErrInsufficientStock
	case errors.Is(err, inventorydomain.ErrLocationNotFound):
		return salesdomain.ErrLocationNotFound
	case errors.Is(err, inventorydomain.ErrLocationInactive):
		return salesdomain.ErrLocationInactive
	case errors.Is(err, inventorydomain.ErrVariantNotFound):
		return salesdomain.ErrProductNotFound
	}
	return err
}
