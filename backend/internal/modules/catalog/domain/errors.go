package domain

import "errors"

// Domain errors of the catalog module. The HTTP adapter maps them to status codes.
var (
	ErrInvalidTenantID       = errors.New("catalog: invalid tenant id")
	ErrInvalidProductID      = errors.New("catalog: invalid product id")
	ErrInvalidCategoryID     = errors.New("catalog: invalid category id")
	ErrInvalidSKU            = errors.New("catalog: invalid SKU")
	ErrInvalidBarcode        = errors.New("catalog: invalid barcode")
	ErrInvalidProductName    = errors.New("catalog: invalid product name")
	ErrInvalidDescription    = errors.New("catalog: invalid product description")
	ErrInvalidUnit           = errors.New("catalog: invalid unit of measure")
	ErrInvalidPrice          = errors.New("catalog: invalid price")
	ErrInvalidCost           = errors.New("catalog: invalid cost")
	ErrInvalidTaxRate        = errors.New("catalog: invalid tax rate")
	ErrInvalidCategoryName   = errors.New("catalog: invalid category name")
	ErrInvalidCategoryParent = errors.New("catalog: invalid category parent")
	ErrProductInactive       = errors.New("catalog: product is inactive")

	// Errors reported by the repositories and the use cases.
	ErrProductNotFound  = errors.New("catalog: product not found")
	ErrCategoryNotFound = errors.New("catalog: category not found")
	ErrSKUTaken         = errors.New("catalog: SKU already used in the tenant")
	ErrBarcodeTaken     = errors.New("catalog: barcode already used in the tenant")
	ErrCategoryInUse    = errors.New("catalog: category has products or subcategories")
	ErrCategoryCycle    = errors.New("catalog: category would be its own ancestor")
)
