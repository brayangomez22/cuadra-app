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
)
