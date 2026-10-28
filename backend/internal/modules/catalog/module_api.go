// The part of the catalog module other modules read through internal/server, which
// adapts it to their ports. The callers already checked the user's
// permissions for their own operation.

package catalog

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// MaxProductsByID is the most products ProductsByID returns in one call.
const MaxProductsByID = 100

// ProductsByID returns the products of tenantID among ids (at most
// MaxProductsByID, distinct), active or not; the ones that do not exist in
// the tenant are left out. It is for other modules (sales reads names,
// prices and IVA through it); internal/server adapts it to their ports. The caller
// already checked the user's permissions for its own operation.
func (m *Module) ProductsByID(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]*domain.Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	if len(ids) > MaxProductsByID {
		return nil, errors.New("catalog: too many products requested")
	}
	page, err := m.service.SearchProducts(ctx, app.Actor{TenantID: tenantID}, domain.ProductFilter{IDs: ids, Limit: len(ids)})
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

// ProductsOfVariants returns the products that the variants among
// variantIDs (at most MaxProductsByID, distinct) belong to in tenantID, with
// all their variants; the variants that do not exist in the tenant are left
// out. It is for other modules that sell variants (sales reads their names,
// labels, prices and IVA through it).
func (m *Module) ProductsOfVariants(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) ([]*domain.Product, error) {
	if len(variantIDs) > MaxProductsByID {
		return nil, errors.New("catalog: too many variants requested")
	}
	productOf, err := m.VariantProducts(ctx, tenantID, variantIDs)
	if err != nil {
		return nil, err
	}
	productIDs := make([]uuid.UUID, 0, len(productOf))
	for _, id := range productOf {
		if !slices.Contains(productIDs, id) {
			productIDs = append(productIDs, id)
		}
	}
	return m.ProductsByID(ctx, tenantID, productIDs)
}

// VariantProducts returns the product of each of variantIDs that exists in
// tenantID; the others are left out. It is for other modules that only know
// the variant (the inventory fills its deprecated product_id with it).
func (m *Module) VariantProducts(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	return m.products.VariantProducts(ctx, tenantID, variantIDs)
}

// VariantsInCategories returns the variants, active or not, of the products
// in categoryIDs or in their subcategories; domain.ErrCategoryNotFound if a
// category does not exist in tenantID. It is for the inventory, which counts
// the stock of some categories.
func (m *Module) VariantsInCategories(ctx context.Context, tenantID uuid.UUID, categoryIDs []uuid.UUID) ([]uuid.UUID, error) {
	return m.products.VariantsInCategories(ctx, tenantID, categoryIDs)
}
