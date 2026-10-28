package domain

import (
	"strings"

	"github.com/shopspring/decimal"
)

// Group is a product of the file: its first row's product data and one
// variant per row.
type Group struct {
	// Row is the number of the product's first row.
	Row      int
	Product  ProductFields
	Variants []VariantLine
	// Invalid reports that one of its rows had errors: the group is not
	// checked against the catalog, the file is already rejected.
	Invalid bool
}

// VariantLine is a variant of a group and its row.
type VariantLine struct {
	Row int
	VariantFields
	Stock    decimal.Decimal
	HasStock bool
}

// Grouper groups the rows of a file, in order, into products: consecutive
// rows with the same product name (without regard to case or spaces) are
// one product with one variant per row. It keeps only the current product
// and the names and SKUs seen, so a file is read one row at a time.
type Grouper struct {
	variants bool
	current  *Group
	key      string
	// products and skus map each name and SKU seen to its first row.
	products map[string]int
	skus     map[string]int
}

// NewGrouper returns a Grouper; with variants false (the module off), a
// product takes one row only.
func NewGrouper(variants bool) *Grouper {
	return &Grouper{variants: variants, products: map[string]int{}, skus: map[string]int{}}
}

// Add adds the next row. valid is false when the row had errors. When the
// row starts another product, the previous one is returned (ok). errs are
// the row's grouping errors: a product whose rows are not consecutive, a
// SKU repeated in the file, a second row without the variants module.
func (g *Grouper) Add(r Row, valid bool) (done Group, ok bool, errs []RowError) {
	if r.Blank {
		return Group{}, false, nil
	}
	errs = g.checkSKU(r)
	key := productKey(r.Product.Name)
	if g.current != nil && key == g.key {
		if !g.variants {
			return Group{}, false, append(errs, RowError{Row: r.Number, Column: ColumnProduct, Code: CodeVariantsInactive})
		}
		g.current.add(r, valid && len(errs) == 0)
		return Group{}, false, errs
	}
	if _, seen := g.products[key]; seen && key != "" {
		errs = append(errs, RowError{Row: r.Number, Column: ColumnProduct, Code: CodeProductSplit})
	}
	errs = append(errs, r.ProductErrors...)
	done, ok = g.Flush()
	g.products[key], g.key = r.Number, key
	g.current = &Group{Row: r.Number, Product: r.Product}
	g.current.add(r, valid && len(errs) == 0)
	return done, ok, errs
}

// checkSKU reports r's SKU if an earlier row has it.
func (g *Grouper) checkSKU(r Row) []RowError {
	sku := strings.ToUpper(strings.TrimSpace(r.Variant.SKU))
	if sku == "" {
		return nil
	}
	if _, seen := g.skus[sku]; seen {
		return []RowError{{Row: r.Number, Column: ColumnSKU, Code: CodeDuplicateSKU}}
	}
	g.skus[sku] = r.Number
	return nil
}

// Flush returns the last product, if any.
func (g *Grouper) Flush() (Group, bool) {
	if g.current == nil {
		return Group{}, false
	}
	done := *g.current
	g.current = nil
	return done, true
}

func (gr *Group) add(r Row, valid bool) {
	gr.Variants = append(gr.Variants, VariantLine{
		Row: r.Number, VariantFields: r.Variant, Stock: r.Stock, HasStock: r.HasStock,
	})
	gr.Invalid = gr.Invalid || !valid
}

// productKey is how product names are compared: in lowercase, with single
// spaces.
func productKey(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(name)), " ")
}
