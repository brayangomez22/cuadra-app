package app_test

import (
	"sync"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// In-memory fakes of the catalog ports. They copy entities in and out, like
// a database would, and enforce the same per-tenant constraints.

type store struct {
	mu         sync.Mutex
	products   map[uuid.UUID]*domain.Product
	categories map[uuid.UUID]*domain.Category
	units      map[uuid.UUID]*domain.UnitOfMeasure
	attributes map[uuid.UUID]*domain.AttributeDefinition
	axes       map[uuid.UUID]*domain.VariantAxis
	// searches records the filters received by Search.
	searches []domain.ProductFilter
}

func cloneProduct(p *domain.Product) *domain.Product {
	variants := make([]domain.VariantSnapshot, 0, len(p.Variants()))
	for _, v := range p.Variants() {
		variants = append(variants, domain.VariantSnapshot{
			ID: v.ID(), SKU: v.SKU(), Barcode: v.Barcode(), Cost: v.Cost(), Price: v.Price(),
			AxisValues: v.AxisValues(), IsDefault: v.IsDefault(), Active: v.IsActive(), CreatedAt: v.CreatedAt(),
			Units: unitParams(v),
		})
	}
	c, err := domain.RehydrateProduct(domain.ProductSnapshot{
		ID: p.ID(), TenantID: p.TenantID(), Name: p.Name(),
		Description: p.Description(), CategoryID: p.CategoryID(), BaseUnit: p.BaseUnit(),
		TaxRate: p.TaxRate(), Attributes: p.Attributes(), Active: p.IsActive(), CreatedAt: p.CreatedAt(),
		Variants: variants,
	})
	if err != nil {
		panic(err)
	}
	return c
}

func unitParams(v *domain.ProductVariant) []domain.VariantUnitParams {
	var out []domain.VariantUnitParams
	for _, u := range v.Units() {
		out = append(out, domain.VariantUnitParams{Unit: u.Unit(), Factor: u.Factor(), Price: u.Price(), Barcode: u.Barcode()})
	}
	return out
}

// barcodes returns every barcode of p: its variants' and their units'.
func barcodes(p *domain.Product) []string {
	var out []string
	for _, v := range p.Variants() {
		if v.Barcode() != "" {
			out = append(out, v.Barcode())
		}
		for _, u := range v.Units() {
			if u.Barcode() != "" {
				out = append(out, u.Barcode())
			}
		}
	}
	return out
}
