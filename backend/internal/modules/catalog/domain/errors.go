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

	ErrInvalidAttributeID      = errors.New("catalog: invalid attribute id")
	ErrInvalidAttributeKey     = errors.New("catalog: invalid attribute key")
	ErrInvalidAttributeLabel   = errors.New("catalog: invalid attribute label")
	ErrInvalidAttributeType    = errors.New("catalog: invalid attribute type")
	ErrInvalidAttributeOptions = errors.New("catalog: invalid attribute options")
	// ErrUnknownAttribute, ErrAttributeRequired and ErrInvalidAttributeValue
	// come wrapped in an *AttributeError that names the attribute.
	ErrUnknownAttribute         = errors.New("catalog: attribute not defined in the tenant")
	ErrAttributeRequired        = errors.New("catalog: required attribute missing")
	ErrInvalidAttributeValue    = errors.New("catalog: invalid attribute value")
	ErrInvalidAxisID            = errors.New("catalog: invalid variant axis id")
	ErrInvalidAxisName          = errors.New("catalog: invalid variant axis name")
	ErrInvalidAxisAllowedValues = errors.New("catalog: invalid variant axis allowed values")
	// ErrUnknownAxis and ErrAxisValueNotAllowed come wrapped in an
	// *AxisError that names the axis.
	ErrUnknownAxis         = errors.New("catalog: variant axis not defined in the tenant")
	ErrAxisValueNotAllowed = errors.New("catalog: value not allowed by the variant axis")

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

	ErrAttributeNotFound    = errors.New("catalog: attribute definition not found")
	ErrAttributeKeyTaken    = errors.New("catalog: attribute key already used in the tenant")
	ErrAttributeInUse       = errors.New("catalog: attribute is used by products")
	ErrAttributeOptionInUse = errors.New("catalog: a removed attribute option is used by products")
	ErrAxisNotFound         = errors.New("catalog: variant axis not found")
	ErrAxisNameTaken        = errors.New("catalog: variant axis name already used in the tenant")
	ErrAxisInUse            = errors.New("catalog: variant axis is used by variants")
	ErrAxisValueInUse       = errors.New("catalog: a removed axis value is used by variants")
)
