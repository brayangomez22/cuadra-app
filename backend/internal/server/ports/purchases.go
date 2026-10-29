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
	purchasesapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	purchasesdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

// PurchasesCatalog satisfies purchasesapp.VariantCatalog with the catalog
// module.
type PurchasesCatalog struct{ Catalog *catalog.Module }

// Variants returns what a purchase needs of each variant of tenantID among
// ids: its names, its product's VAT rate and base unit, and its alternate
// units.
func (c PurchasesCatalog) Variants(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]purchasesapp.VariantInfo, error) {
	products, err := c.Catalog.ProductsOfVariants(ctx, tenantID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]purchasesapp.VariantInfo, 0, len(ids))
	for _, p := range products {
		for _, v := range p.Variants() {
			if !slices.Contains(ids, v.ID()) {
				continue
			}
			info := purchasesapp.VariantInfo{
				ID: v.ID(), ProductName: p.Name(), Label: v.Label(), BaseUnitCode: p.BaseUnit().Code,
				TaxRate: p.TaxRate().Rate(), Active: p.IsActive() && v.IsActive(),
			}
			for _, u := range v.Units() {
				info.Units = append(info.Units, purchasesapp.UnitInfo{UnitID: u.Unit().ID, Code: u.Unit().Code, Factor: u.Factor()})
			}
			out = append(out, info)
		}
	}
	return out, nil
}

// PurchasesStock satisfies purchasesapp.PurchaseStock with the inventory
// module: a purchase's lines enter the stock of its location.
type PurchasesStock struct{ Inventory *inventory.Module }

// ReceivePurchase registers the entries of the purchase's lines, in its
// transaction, and returns what each put into the stock.
func (s PurchasesStock) ReceivePurchase(ctx context.Context, tenantID, userID, locationID, purchaseID uuid.UUID, lines []purchasesapp.StockLine) ([]purchasesdomain.Received, error) {
	in := inventoryapp.PurchaseStockInput{LocationID: locationID, PurchaseID: purchaseID}
	for _, l := range lines {
		in.Lines = append(in.Lines, inventoryapp.PurchaseStockLine{
			VariantID: l.VariantID, Quantity: l.Quantity, UnitCost: l.UnitCost, Factor: l.Factor,
		})
	}
	movements, err := s.Inventory.ReceivePurchaseStock(ctx, tenantID, userID, in)
	if err != nil {
		return nil, purchasesStockError(err, lines)
	}
	out := make([]purchasesdomain.Received, len(movements))
	for i, m := range movements {
		out[i] = purchasesdomain.Received{BaseQuantity: m.Quantity(), BaseUnitCost: m.UnitCost()}
	}
	return out, nil
}

// purchasesStockError translates the inventory's errors into the purchases
// domain errors the purchases module maps to HTTP. An error of one variant
// names its first line. Others pass through as unexpected.
func purchasesStockError(err error, lines []purchasesapp.StockLine) error {
	var mapped error
	switch {
	case errors.Is(err, inventorydomain.ErrLocationNotFound):
		return purchasesdomain.ErrLocationNotFound
	case errors.Is(err, inventorydomain.ErrLocationInactive):
		return purchasesdomain.ErrLocationInactive
	case errors.Is(err, inventorydomain.ErrQuantityPrecision):
		mapped = purchasesdomain.ErrQuantityPrecision
	case errors.Is(err, inventorydomain.ErrVariantNotFound):
		mapped = purchasesdomain.ErrVariantNotFound
	case errors.Is(err, inventorydomain.ErrInvalidQuantity):
		mapped = purchasesdomain.ErrInvalidQuantity
	default:
		return err
	}
	var variantErr *inventoryapp.VariantStockError
	if errors.As(err, &variantErr) {
		i := slices.IndexFunc(lines, func(l purchasesapp.StockLine) bool { return l.VariantID == variantErr.VariantID })
		if i >= 0 {
			return &purchasesapp.LineError{Index: i, Err: mapped}
		}
	}
	return mapped
}
