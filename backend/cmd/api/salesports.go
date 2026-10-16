package main

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory"
	inventoryapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	inventorydomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	salesapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	salesdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// The sales module reaches the catalog and the inventory through its own
// ports (salesapp.ProductCatalog, salesapp.StockReserver), and the inventory
// reaches the catalog through inventoryapp.ProductVariants; these adapters
// satisfy them with the other modules and translate their errors, so no
// module imports another.

// salesCatalog satisfies salesapp.ProductCatalog with the catalog module.
type salesCatalog struct{ m *catalog.Module }

func (c salesCatalog) Products(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]salesapp.ProductInfo, error) {
	products, err := c.m.ProductsByID(ctx, tenantID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]salesapp.ProductInfo, 0, len(products))
	for _, p := range products {
		// Sales still sells products (T24c moves it to variants): it sells the
		// default variant, at its price, while both are active.
		v := p.DefaultVariant()
		out = append(out, salesapp.ProductInfo{
			ID: p.ID(), Name: p.Name(), Price: v.Price(), TaxRate: p.TaxRate().Rate(), Active: p.IsActive() && v.IsActive(),
		})
	}
	return out, nil
}

// salesStock satisfies salesapp.StockReserver with the inventory module.
// Sales still sells products (T24c moves it to variants), and the inventory
// counts variants: each product's stock goes out of its default variant.
type salesStock struct {
	inventory *inventory.Module
	catalog   *catalog.Module
}

func (s salesStock) ReserveSale(ctx context.Context, tenantID, userID, locationID, saleID uuid.UUID, lines []salesapp.StockLine) error {
	productIDs := make([]uuid.UUID, 0, len(lines))
	for _, l := range lines {
		productIDs = append(productIDs, l.ProductID)
	}
	// Within the sale's transaction (ctx), like the exits.
	products, err := s.catalog.ProductsByID(ctx, tenantID, productIDs)
	if err != nil {
		return err
	}
	variantOf := make(map[uuid.UUID]uuid.UUID, len(products))
	productOf := make(map[uuid.UUID]uuid.UUID, len(products))
	for _, p := range products {
		v := p.DefaultVariant().ID()
		variantOf[p.ID()], productOf[v] = v, p.ID()
	}
	in := inventoryapp.SaleStockInput{LocationID: locationID, SaleID: saleID}
	for _, l := range lines {
		variantID, ok := variantOf[l.ProductID]
		if !ok {
			return salesdomain.ErrProductNotFound
		}
		in.Lines = append(in.Lines, inventoryapp.SaleStockLine{VariantID: variantID, Quantity: l.Quantity})
	}
	return salesStockError(s.inventory.IssueSaleStock(ctx, tenantID, userID, in), productOf)
}

func (s salesStock) ReverseSale(ctx context.Context, tenantID, userID, saleID uuid.UUID) error {
	return salesStockError(s.inventory.ReverseSaleStock(ctx, tenantID, userID, saleID), nil)
}

// salesStockError translates the inventory's errors into the sales domain
// errors the sales module maps to HTTP; productOf names the product of each
// variant of the sale. Others pass through as unexpected.
func salesStockError(err error, productOf map[uuid.UUID]uuid.UUID) error {
	var variantErr *inventoryapp.VariantStockError
	switch {
	case err == nil:
		return nil
	case errors.Is(err, inventorydomain.ErrInsufficientStock) && errors.As(err, &variantErr) && productOf[variantErr.VariantID] != uuid.Nil:
		return &salesdomain.InsufficientStockError{ProductID: productOf[variantErr.VariantID]}
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
// module, for the deprecated product_id of the inventory contract.
type inventoryCatalog struct{ m *catalog.Module }

func (c inventoryCatalog) DefaultVariant(ctx context.Context, tenantID, productID uuid.UUID) (uuid.UUID, error) {
	products, err := c.m.ProductsByID(ctx, tenantID, []uuid.UUID{productID})
	if err != nil {
		return uuid.Nil, err
	}
	if len(products) == 0 {
		return uuid.Nil, inventorydomain.ErrProductNotFound
	}
	return products[0].DefaultVariant().ID(), nil
}

func (c inventoryCatalog) ProductsOf(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	return c.m.VariantProducts(ctx, tenantID, variantIDs)
}
