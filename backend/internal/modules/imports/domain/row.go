package domain

import "github.com/shopspring/decimal"

// Row is one data row of an import file: a variant and, in the first row
// of its product, the product's data.
type Row struct {
	// Number is the row's number in the file, counting the header as 1.
	Number int
	// Blank reports a row with every cell empty, which is skipped.
	Blank   bool
	Product ProductFields
	Variant VariantFields
	// Stock is the initial stock at the import's location, in the
	// product's base unit, when HasStock.
	Stock    decimal.Decimal
	HasStock bool
	// ProductErrors are the required product values the row leaves empty
	// (its unit, a required attribute). They only count when the row starts
	// a product: the next rows of a product may leave them empty.
	ProductErrors []RowError
}

// ProductFields are the product's columns of a row. Category and Unit name
// the tenant's category and unit (by name and by code).
type ProductFields struct {
	Name        string
	Description string
	Category    string
	Unit        string
	TaxRate     decimal.Decimal
	// Attributes are the values given, by attribute key.
	Attributes map[string]AttributeValue
}

// AttributeValue is the value of an attribute column: a text for text,
// number and list attributes (the catalog checks it against the
// definition), a boolean for yes/no ones.
type AttributeValue struct {
	Text   string
	Bool   bool
	IsBool bool
}

// VariantFields are the variant's columns of a row. Axes are the values by
// axis name.
type VariantFields struct {
	SKU     string
	Barcode string
	Cost    decimal.Decimal
	Price   decimal.Decimal
	// PriceText is the price cell as written when it is not a valid amount,
	// so the preview shows what the file says; empty otherwise.
	PriceText string
	Axes      map[string]string
}

// RowError is a problem of one row (Row 1 is the header). Column names the
// column, if the problem is in one. Code is stable; Message, when set,
// already describes the problem (the catalog's reasons come with theirs),
// and otherwise the adapter that shows the error writes it from the code.
type RowError struct {
	Row     int
	Column  string
	Code    string
	Message string
}

// The codes of the problems the import finds on its own.
const (
	CodeMissingColumn    = "missing_column"
	CodeUnknownColumn    = "unknown_column"
	CodeDuplicateColumn  = "duplicate_column"
	CodeEmptyFile        = "empty_file"
	CodeTooManyRows      = "too_many_rows"
	CodeRequired         = "required"
	CodeNotANumber       = "not_a_number"
	CodeNegativeNumber   = "negative_number"
	CodeInvalidBoolean   = "invalid_boolean"
	CodeDuplicateSKU     = "duplicate_sku"
	CodeProductSplit     = "product_split"
	CodeVariantsInactive = "variants_inactive"
)

// MaxErrors is how many row errors an import keeps; the count goes on.
const MaxErrors = 1000
