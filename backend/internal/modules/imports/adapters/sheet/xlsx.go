package sheet

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// xlsxRows reads the first sheet of a workbook. The cells are read raw, as
// stored (12500.5 and not "$ 12.500,50"), so the numbers are plain.
func xlsxRows(data []byte) (app.Sheet, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{RawCellValue: true})
	if err != nil {
		return app.Sheet{}, fmt.Errorf("%w: %w", domain.ErrUnreadableFile, err)
	}
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		_ = f.Close()
		return app.Sheet{}, fmt.Errorf("%w: no sheets", domain.ErrUnreadableFile)
	}
	rows := func(yield func([]string, error) bool) {
		defer func() { _ = f.Close() }()
		it, err := f.Rows(sheets[0])
		if err != nil {
			yield(nil, err)
			return
		}
		defer func() { _ = it.Close() }()
		for it.Next() {
			cells, err := it.Columns()
			if !yield(cells, err) || err != nil {
				return
			}
		}
		if err := it.Error(); err != nil {
			yield(nil, err)
		}
	}
	return app.Sheet{Rows: rows, Style: domain.PlainNumbers}, nil
}
