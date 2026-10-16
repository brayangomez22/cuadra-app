package domain

import "errors"

// Domain errors of the catalog module. The HTTP adapter maps them to status codes.
var (
	ErrInvalidTenantID    = errors.New("catalog: invalid tenant id")
	ErrInvalidProductID   = errors.New("catalog: invalid product id")
	ErrInvalidCategoryID  = errors.New("catalog: invalid category id")
	ErrInvalidSKU         = errors.New("catalog: invalid SKU")
	ErrInvalidBarcode     = errors.New("catalog: invalid barcode")
	ErrInvalidProductName = errors.New("catalog: invalid product name")
	ErrInvalidDescription = errors.New("catalog: invalid product description")
	// ErrInvalidUnit reports a product without a base unit.
	ErrInvalidUnit           = errors.New("catalog: invalid unit of measure")
	ErrInvalidUnitID         = errors.New("catalog: invalid unit id")
	ErrInvalidUnitCode       = errors.New("catalog: invalid unit code")
	ErrInvalidUnitName       = errors.New("catalog: invalid unit name")
	ErrInvalidPrice          = errors.New("catalog: invalid price")
	ErrInvalidCost           = errors.New("catalog: invalid cost")
	ErrInvalidTaxRate        = errors.New("catalog: invalid tax rate")
	ErrInvalidCategoryName   = errors.New("catalog: invalid category name")
	ErrInvalidCategoryParent = errors.New("catalog: invalid category parent")
	ErrProductInactive       = errors.New("catalog: product is inactive")
	ErrInvalidVariantID      = errors.New("catalog: invalid variant id")
	ErrInvalidAxisValues     = errors.New("catalog: invalid variant axis values")
	// ErrInvalidVariants reports a stored product without exactly one
	// default variant.
	ErrInvalidVariants     = errors.New("catalog: a product needs exactly one default variant")
	ErrVariantAxesMismatch = errors.New("catalog: every variant of a product must name the same axes")
	ErrDuplicateVariant    = errors.New("catalog: another variant of the product has the same axis values")
	ErrVariantNotFound     = errors.New("catalog: variant not found")

	// Errors reported by the repositories and the use cases.
	ErrProductNotFound  = errors.New("catalog: product not found")
	ErrCategoryNotFound = errors.New("catalog: category not found")
	ErrSKUTaken         = errors.New("catalog: SKU already used in the tenant")
	ErrBarcodeTaken     = errors.New("catalog: barcode already used in the tenant")
	ErrCategoryInUse    = errors.New("catalog: category has products or subcategories")
	ErrCategoryCycle    = errors.New("catalog: category would be its own ancestor")
	ErrUnitNotFound     = errors.New("catalog: unit of measure not found")
	ErrUnitCodeTaken    = errors.New("catalog: unit code already used in the tenant")
	ErrUnitInUse        = errors.New("catalog: unit of measure is used by products")
	// ErrUnitRequired reports a product input with neither a unit id nor a
	// unit code, or with both.
	ErrUnitRequired = errors.New("catalog: a product needs a unit id or a unit code, not both")
)
