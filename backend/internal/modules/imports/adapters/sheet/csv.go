package sheet

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// bom starts a UTF-8 file saved by a spreadsheet.
var bom = []byte{0xEF, 0xBB, 0xBF}

// csvRows reads a CSV file. A spreadsheet in Spanish saves it separated by
// ";" with a decimal comma, one in English by "," with a decimal point: the
// header row tells which. A file that is not UTF-8 is read as Latin-1, the
// encoding a spreadsheet on Windows saves "CSV" in.
func csvRows(data []byte) app.Sheet {
	data = bytes.TrimPrefix(data, bom)
	if !utf8.Valid(data) {
		data = latin1ToUTF8(data)
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma, r.FieldsPerRecord, r.LazyQuotes = ',', -1, true
	style := domain.PlainNumbers
	if header, _, _ := bytes.Cut(data, []byte("\n")); bytes.Count(header, []byte(";")) > bytes.Count(header, []byte(",")) {
		r.Comma, style = ';', domain.DecimalComma
	}
	rows := func(yield func([]string, error) bool) {
		for {
			record, err := r.Read()
			if errors.Is(err, io.EOF) {
				return
			}
			if !yield(record, err) || err != nil {
				return
			}
		}
	}
	return app.Sheet{Rows: rows, Style: style}
}

// latin1ToUTF8 decodes Latin-1: each byte is the code point of its value.
func latin1ToUTF8(data []byte) []byte {
	var b strings.Builder
	b.Grow(len(data) + len(data)/8)
	for _, c := range data {
		b.WriteRune(rune(c))
	}
	return []byte(b.String())
}

// csvTemplate writes the header row separated by ";" and with a BOM, as a
// spreadsheet in Spanish opens it with each column in its own cell.
func csvTemplate(l domain.Layout) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(bom)
	w := csv.NewWriter(&buf)
	w.Comma, w.UseCRLF = ';', true
	if err := w.Write(l.Headers()); err != nil {
		return nil, err
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
