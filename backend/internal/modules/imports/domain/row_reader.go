package domain

import (
	"strings"

	"github.com/shopspring/decimal"
)

// rowReader reads the cells of one data row and collects its errors.
type rowReader struct {
	header Header
	cells  []string
	style  NumberStyle
	number int
	errs   []RowError
	// productErrs are the required product values missing (Row.ProductErrors).
	productErrs []RowError
}

func (r *rowReader) fail(column, code string) {
	r.errs = append(r.errs, RowError{Row: r.number, Column: column, Code: code})
}

// text returns the trimmed cell of column, "" if the file has no such
// column.
func (r *rowReader) text(column string) string {
	i, ok := r.header.index[column]
	if !ok || i >= len(r.cells) {
		return ""
	}
	return strings.TrimSpace(r.cells[i])
}

// required returns the cell of column, reporting it when empty.
func (r *rowReader) required(column string) string {
	s := r.text(column)
	if s == "" {
		r.fail(column, CodeRequired)
	}
	return s
}

// productValue returns the cell of a required product column, reporting it
// in productErrs when empty.
func (r *rowReader) productValue(column string) string {
	s := r.text(column)
	if s == "" {
		r.productErrs = append(r.productErrs, RowError{Row: r.number, Column: column, Code: CodeRequired})
	}
	return s
}

// amount reads a number that cannot be negative; ok is false when the cell
// is empty or has an error (already reported).
func (r *rowReader) amount(column string) (_ decimal.Decimal, ok bool) {
	s := r.text(column)
	if s == "" {
		return decimal.Zero, false
	}
	d, ok := ParseNumber(s, r.style)
	switch {
	case !ok:
		r.fail(column, CodeNotANumber)
	case d.IsNegative():
		r.fail(column, CodeNegativeNumber)
	default:
		return d, true
	}
	return decimal.Zero, false
}

func (r *rowReader) product() ProductFields {
	return ProductFields{
		Name:        r.required(ColumnProduct),
		Description: r.text(ColumnDescription),
		Category:    r.text(ColumnCategory),
		Unit:        r.productValue(ColumnUnit),
		TaxRate:     r.taxRate(),
		Attributes:  r.attributes(),
	}
}

// taxRate reads the IVA, or the tenant's when the cell is empty.
func (r *rowReader) taxRate() decimal.Decimal {
	s := r.text(ColumnTaxRate)
	if s == "" {
		return r.header.layout.DefaultTaxRate
	}
	d, ok := ParseTaxRate(s, r.style)
	if !ok {
		r.fail(ColumnTaxRate, CodeNotANumber)
	}
	return d
}

func (r *rowReader) variant() VariantFields {
	v := VariantFields{SKU: r.required(ColumnSKU), Barcode: r.text(ColumnBarcode)}
	if r.text(ColumnPrice) == "" {
		r.fail(ColumnPrice, CodeRequired)
	}
	if price, ok := r.amount(ColumnPrice); ok {
		v.Price = price
	} else {
		v.PriceText = r.text(ColumnPrice)
	}
	v.Cost, _ = r.amount(ColumnCost)
	for _, axis := range r.header.layout.Axes {
		if value := r.text(AxisPrefix + axis.Name); value != "" {
			if v.Axes == nil {
				v.Axes = map[string]string{}
			}
			v.Axes[axis.Name] = value
		}
	}
	return v
}

// attributes reads the attribute columns by their definition's kind; the
// required ones left empty go to productErrs.
func (r *rowReader) attributes() map[string]AttributeValue {
	out := map[string]AttributeValue{}
	for _, a := range r.header.layout.Attributes {
		column := AttributePrefix + a.Label
		s := r.text(column)
		if a.Required {
			s = r.productValue(column)
		}
		if s == "" {
			continue
		}
		if v, ok := r.attribute(a.Type, column, s); ok {
			out[a.Key] = v
		}
	}
	return out
}

func (r *rowReader) attribute(kind AttributeKind, column, s string) (AttributeValue, bool) {
	switch kind {
	case KindNumber:
		d, ok := ParseNumber(s, r.style)
		if !ok {
			r.fail(column, CodeNotANumber)
		}
		return AttributeValue{Text: d.String()}, ok
	case KindBoolean:
		b, ok := parseYesNo(s)
		if !ok {
			r.fail(column, CodeInvalidBoolean)
		}
		return AttributeValue{Bool: b, IsBool: true}, ok
	default:
		return AttributeValue{Text: s}, true
	}
}

// parseYesNo reads a yes/no cell: sí, si, s, x, 1, true, yes or no, n, 0,
// false, without regard to case.
func parseYesNo(s string) (value, ok bool) {
	switch strings.ToLower(s) {
	case "sí", "si", "s", "x", "1", "true", "yes":
		return true, true
	case "no", "n", "0", "false":
		return false, true
	}
	return false, false
}
