package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

// UnitInfo is an alternate unit of a variant: how many base units it holds.
type UnitInfo struct {
	UnitID uuid.UUID
	Code   string
	Factor decimal.Decimal
}

// VariantInfo is what a purchase needs of a variant: its names, its
// product's VAT rate and base unit, and its alternate units.
type VariantInfo struct {
	ID           uuid.UUID
	ProductName  string
	Label        string
	BaseUnitCode string
	TaxRate      decimal.Decimal
	Active       bool
	Units        []UnitInfo
}

// VariantCatalog reads the catalog's variants. Variants returns those of
// tenantID among ids; a missing one is not the tenant's.
type VariantCatalog interface {
	Variants(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]VariantInfo, error)
}

// StockLine is what a purchase line puts into the stock: a quantity in a
// unit holding Factor base units, at UnitCost per unit.
type StockLine struct {
	VariantID uuid.UUID
	Quantity  decimal.Decimal
	UnitCost  decimal.Decimal
	Factor    decimal.Decimal
}

// PurchaseStock is the inventory, as purchases see it.
type PurchaseStock interface {
	// ReceivePurchase registers one entry per line at the location,
	// referencing the purchase and in the purchase's transaction (through
	// ctx), and returns what each line put into the stock, in the base unit
	// and in line order. It fails with the purchases domain's errors
	// (ErrLocationNotFound, ErrLocationInactive, ErrQuantityPrecision...).
	ReceivePurchase(ctx context.Context, tenantID, userID, locationID, purchaseID uuid.UUID, lines []StockLine) ([]domain.Received, error)
}
