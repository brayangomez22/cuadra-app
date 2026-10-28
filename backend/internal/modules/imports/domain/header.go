package domain

import (
	"slices"
	"strings"
)

// Header is the header row of a file: which column each cell is.
type Header struct {
	layout  Layout
	columns []string
	// index maps each column's canonical name to its cell.
	index map[string]int
}

// ParseHeader reads the header row against the tenant's layout. Column
// names are matched without regard to case, spaces or accents. Its errors
// are in row 1: a required column missing, an unknown or repeated one.
func ParseHeader(l Layout, cells []string) (Header, []RowError) {
	known := map[string]string{}
	for _, name := range l.Headers() {
		known[columnKey(name)] = name
	}
	h := Header{layout: l, columns: make([]string, len(cells)), index: map[string]int{}}
	var errs []RowError
	for i, cell := range cells {
		if strings.TrimSpace(cell) == "" {
			continue
		}
		name, ok := known[columnKey(cell)]
		switch _, repeated := h.index[name]; {
		case !ok:
			errs = append(errs, RowError{Row: 1, Column: strings.TrimSpace(cell), Code: CodeUnknownColumn})
		case repeated:
			errs = append(errs, RowError{Row: 1, Column: name, Code: CodeDuplicateColumn})
		default:
			h.columns[i], h.index[name] = name, i
		}
	}
	for _, name := range requiredColumns {
		if _, ok := h.index[name]; !ok {
			errs = append(errs, RowError{Row: 1, Column: name, Code: CodeMissingColumn})
		}
	}
	return h, errs
}

// columnKey is how a column name is compared: in lowercase, without
// accents, with single spaces and one space after a colon.
func columnKey(name string) string {
	name = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", ":", ": ").
		Replace(strings.ToLower(name))
	return strings.Join(strings.Fields(name), " ")
}

// Columns returns the canonical name of each column of the file, in its
// order ("" for an empty or rejected header cell).
func (h Header) Columns() []string { return slices.Clone(h.columns) }

// Parse reads a data row. It reports each cell that cannot be read and each
// required value missing; the catalog checks the rest (SKU format, units,
// attribute options...).
func (h Header) Parse(number int, cells []string, style NumberStyle) (Row, []RowError) {
	if !slices.ContainsFunc(cells, func(c string) bool { return strings.TrimSpace(c) != "" }) {
		return Row{Number: number, Blank: true}, nil
	}
	r := &rowReader{header: h, cells: cells, style: style, number: number}
	row := Row{
		Number:  number,
		Product: r.product(),
		Variant: r.variant(),
	}
	row.Stock, row.HasStock = r.amount(ColumnStock)
	row.ProductErrors = r.productErrs
	return row, r.errs
}
