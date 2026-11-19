package sheet

import (
	"bytes"
	"encoding/csv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// bom makes a spreadsheet read the CSV as UTF-8, with its accents.
var bom = []byte("\xEF\xBB\xBF")

// writeCSV writes t separated by ";" with a decimal comma, as a
// spreadsheet in Spanish opens it with each column in its own cell.
func writeCSV(t Table) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(bom)
	w := csv.NewWriter(&buf)
	w.Comma, w.UseCRLF = ';', true
	if err := w.Write(t.Columns); err != nil {
		return nil, err
	}
	for _, row := range t.Rows {
		record := make([]string, len(row))
		for i, c := range row {
			record[i] = c.text
			if c.number != nil {
				record[i] = strings.Replace(c.number.String(), ".", ",", 1)
			}
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// writeXLSX writes t in a sheet named after it, with a bold header row.
// Figures go as numbers (not text), so the spreadsheet adds them up.
func writeXLSX(t Table) (_ []byte, err error) {
	f := excelize.NewFile()
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	if err := f.SetSheetName(f.GetSheetName(0), t.Name); err != nil {
		return nil, err
	}
	if err := writeHeader(f, t); err != nil {
		return nil, err
	}
	for r, row := range t.Rows {
		if err := writeRow(f, t.Name, r+2, row); err != nil {
			return nil, err
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeHeader(f *excelize.File, t Table) error {
	cells := make([]any, len(t.Columns))
	for i, c := range t.Columns {
		cells[i] = c
	}
	if err := f.SetSheetRow(t.Name, "A1", &cells); err != nil {
		return err
	}
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}
	last, err := excelize.CoordinatesToCellName(max(len(t.Columns), 1), 1)
	if err != nil {
		return err
	}
	return f.SetCellStyle(t.Name, "A1", last, bold)
}

func writeRow(f *excelize.File, sheet string, row int, cells []Cell) error {
	for i, c := range cells {
		name, err := excelize.CoordinatesToCellName(i+1, row)
		if err != nil {
			return err
		}
		if c.number != nil {
			err = f.SetCellDefault(sheet, name, c.number.String())
		} else {
			err = f.SetCellStr(sheet, name, c.text)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
