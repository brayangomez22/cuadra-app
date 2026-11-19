// Package sheet writes a business report as a spreadsheet: an .xlsx whose
// figures are numbers Excel can add up, or a CSV as a spreadsheet in
// Spanish opens it (";", decimal comma and a BOM). It knows nothing of the
// reports: the HTTP adapter builds the Table.
package sheet

import (
	"errors"

	"github.com/shopspring/decimal"
)

// MaxRows is the most rows an export writes.
const MaxRows = 5000

// Format is the kind of file.
type Format string

// The export formats.
const (
	XLSX Format = "xlsx"
	CSV  Format = "csv"
)

// ErrUnknownFormat is a format other than xlsx or csv.
var ErrUnknownFormat = errors.New("sheet: unknown format")

// Cell is a text or a figure.
type Cell struct {
	text   string
	number *decimal.Decimal
}

// Text is a text cell.
func Text(s string) Cell { return Cell{text: s} }

// Number is a figure cell.
func Number(d decimal.Decimal) Cell { return Cell{number: &d} }

// Count is a whole number cell.
func Count(n int) Cell { return Number(decimal.NewFromInt(int64(n))) }

// Table is a report as a sheet: a name, its column titles and its rows.
type Table struct {
	Name    string
	Columns []string
	Rows    [][]Cell
}

// Add appends a row.
func (t *Table) Add(cells ...Cell) { t.Rows = append(t.Rows, cells) }

// Write returns t as a file of format.
func Write(t Table, format Format) ([]byte, error) {
	switch format {
	case XLSX:
		return writeXLSX(t)
	case CSV:
		return writeCSV(t)
	default:
		return nil, ErrUnknownFormat
	}
}
