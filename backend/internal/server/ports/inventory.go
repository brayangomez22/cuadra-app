package ports

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	catalogdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	inventorydomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// InventoryCatalog satisfies inventoryapp.ProductVariants with the catalog
// module: the product of each variant, for the product_id of the levels and
// movements.
type InventoryCatalog struct{ Catalog *catalog.Module }

// ProductsOf returns the product of each variant of tenantID among
// variantIDs.
func (c InventoryCatalog) ProductsOf(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	return c.Catalog.VariantProducts(ctx, tenantID, variantIDs)
}

// UnitFactor returns the factor of an alternate unit of the variant, from
// the catalog. A variant that does not exist or lacks the unit is
// inventorydomain.ErrUnitNotAvailable.
func (c InventoryCatalog) UnitFactor(ctx context.Context, tenantID, variantID, unitID uuid.UUID) (decimal.Decimal, error) {
	products, err := c.Catalog.ProductsOfVariants(ctx, tenantID, []uuid.UUID{variantID})
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

// VariantsInCategories returns the variants of the products in the
// categories and their subcategories, from the catalog. A category that does
// not exist is inventorydomain.ErrCategoryNotFound.
func (c InventoryCatalog) VariantsInCategories(ctx context.Context, tenantID uuid.UUID, categoryIDs []uuid.UUID) ([]uuid.UUID, error) {
	variants, err := c.Catalog.VariantsInCategories(ctx, tenantID, categoryIDs)
	if errors.Is(err, catalogdomain.ErrCategoryNotFound) {
		return nil, inventorydomain.ErrCategoryNotFound
	}
	return variants, err
}
