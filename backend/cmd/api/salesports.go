package main

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

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

// salesCatalog satisfies salesapp.VariantCatalog with the catalog module.
type salesCatalog struct{ m *catalog.Module }

func (c salesCatalog) Variants(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]salesapp.VariantInfo, error) {
	products, err := c.m.ProductsOfVariants(ctx, tenantID, ids)
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

// salesStock satisfies salesapp.StockReserver with the inventory module: a
// sale takes its stock out of the variants it sold.
type salesStock struct{ inventory *inventory.Module }

func (s salesStock) ReserveSale(ctx context.Context, tenantID, userID, locationID, saleID uuid.UUID, lines []salesapp.StockLine) error {
	in := inventoryapp.SaleStockInput{LocationID: locationID, SaleID: saleID}
	for _, l := range lines {
		in.Lines = append(in.Lines, inventoryapp.SaleStockLine{VariantID: l.VariantID, Quantity: l.Quantity})
	}
	return salesStockError(s.inventory.IssueSaleStock(ctx, tenantID, userID, in))
}

func (s salesStock) ReverseSale(ctx context.Context, tenantID, userID, saleID uuid.UUID) error {
	return salesStockError(s.inventory.ReverseSaleStock(ctx, tenantID, userID, saleID))
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

// inventoryCatalog satisfies inventoryapp.ProductVariants with the catalog
// module: the product of each variant, for the product_id of the levels and
// movements.
type inventoryCatalog struct{ m *catalog.Module }

func (c inventoryCatalog) ProductsOf(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	return c.m.VariantProducts(ctx, tenantID, variantIDs)
}

// UnitFactor returns the factor of an alternate unit of the variant, from
// the catalog. A variant that does not exist or lacks the unit is
// inventorydomain.ErrUnitNotAvailable.
func (c inventoryCatalog) UnitFactor(ctx context.Context, tenantID, variantID, unitID uuid.UUID) (decimal.Decimal, error) {
	products, err := c.m.ProductsOfVariants(ctx, tenantID, []uuid.UUID{variantID})
	if err != nil {
		return decimal.Decimal{}, err
	}
	for _, p := range products {
		v, err := p.Variant(variantID)
		if err != nil {
			continue
		}
		u, err := v.Unit(unitID)
		if err != nil {
			break
		}
		return u.Factor(), nil
	}
	return decimal.Decimal{}, inventorydomain.ErrUnitNotAvailable
}
