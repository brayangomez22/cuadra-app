// Package sheet reads and writes the import files: .xlsx with excelize and
// .csv with encoding/csv. It reads one row at a time, so the rows of a large
// file are never all in memory at once.
package sheet

import (
	"fmt"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// Sheets implements app.Sheets.
type Sheets struct{}

// Rows reads the rows of data in format, header first.
func (Sheets) Rows(format domain.Format, data []byte) (app.Sheet, error) {
	switch format {
	case domain.FormatXLSX:
		return xlsxRows(data)
	case domain.FormatCSV:
		return csvRows(data), nil
	}
	return app.Sheet{}, fmt.Errorf("%w: format %q", domain.ErrUnreadableFile, format)
}

// Template writes an empty file in format with l's columns.
func (Sheets) Template(format domain.Format, l domain.Layout) ([]byte, error) {
	switch format {
	case domain.FormatXLSX:
		return xlsxTemplate(l)
	case domain.FormatCSV:
		return csvTemplate(l)
	}
	return nil, fmt.Errorf("%w: format %q", domain.ErrUnreadableFile, format)
}
