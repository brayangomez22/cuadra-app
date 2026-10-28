package domain

import (
	"slices"

	"github.com/shopspring/decimal"
)

// The columns of an import file, as the template names them. They are part
// of the file format the user fills in, so they are in Spanish.
const (
	ColumnProduct     = "producto"
	ColumnDescription = "descripción"
	ColumnCategory    = "categoría"
	ColumnUnit        = "unidad"
	ColumnTaxRate     = "iva"
	ColumnSKU         = "sku"
	ColumnBarcode     = "código_barras"
	ColumnCost        = "costo"
	ColumnPrice       = "precio"
	ColumnStock       = "stock_inicial"
	// AttributePrefix and AxisPrefix start the column of an attribute (by
	// its label) and of a variant axis (by its name).
	AttributePrefix = "atributo: "
	AxisPrefix      = "eje: "
)

// baseColumns are the columns every file has, in the template's order.
var baseColumns = []string{
	ColumnProduct, ColumnDescription, ColumnCategory, ColumnUnit, ColumnTaxRate,
	ColumnSKU, ColumnBarcode, ColumnCost, ColumnPrice, ColumnStock,
}

// requiredColumns must be in every file; the others may be left out.
var requiredColumns = []string{ColumnProduct, ColumnUnit, ColumnSKU, ColumnPrice}

// AttributeKind is the kind of value an attribute column holds, as the
// catalog defines the attribute.
type AttributeKind string

// The kinds of attribute: free text, a decimal number, one option of a
// list, and yes/no.
const (
	KindText    AttributeKind = "text"
	KindNumber  AttributeKind = "number"
	KindList    AttributeKind = "list"
	KindBoolean AttributeKind = "boolean"
)

// AttributeColumn is a product attribute the tenant defines.
type AttributeColumn struct {
	Key      string
	Label    string
	Type     AttributeKind
	Required bool
	// Options are the values of a list attribute, for the template's help.
	Options []string
}

// AxisColumn is a variant axis the tenant defines, with its values.
type AxisColumn struct {
	Name   string
	Values []string
}

// Layout is the shape of a tenant's import file: the base columns, one per
// attribute and, with the variants module on, one per variant axis.
type Layout struct {
	// Units are the codes of the tenant's units, for the template's help.
	Units      []string
	Attributes []AttributeColumn
	Axes       []AxisColumn
	// Variants reports whether the variants module is on: a product may
	// then take several rows, one per variant.
	Variants bool
	// DefaultTaxRate is the IVA of a row that leaves it empty.
	DefaultTaxRate decimal.Decimal
}

// Headers returns the template's header row.
func (l Layout) Headers() []string {
	h := slices.Clone(baseColumns)
	for _, a := range l.Attributes {
		h = append(h, AttributePrefix+a.Label)
	}
	if l.Variants {
		for _, a := range l.Axes {
			h = append(h, AxisPrefix+a.Name)
		}
	}
	return h
}
