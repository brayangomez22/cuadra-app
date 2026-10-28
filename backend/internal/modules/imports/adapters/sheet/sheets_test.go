package sheet_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/sheet"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

func readAll(t *testing.T, format domain.Format, data []byte) ([][]string, domain.NumberStyle) {
	t.Helper()
	s, err := sheet.Sheets{}.Rows(format, data)
	require.NoError(t, err)
	var rows [][]string
	for cells, err := range s.Rows {
		require.NoError(t, err)
		rows = append(rows, cells)
	}
	return rows, s.Style
}

var clothing = domain.Layout{
	Units:      []string{"und", "par"},
	Attributes: []domain.AttributeColumn{{Key: "material", Label: "Material", Type: domain.KindList, Options: []string{"Algodón", "Cuero"}}},
	Axes:       []domain.AxisColumn{{Name: "talla", Values: []string{"S", "M"}}},
	Variants:   true,
}

func TestCSVRows(t *testing.T) {
	t.Run("un CSV de una hoja en español: punto y coma, BOM y coma decimal", func(t *testing.T) {
		data := append([]byte{0xEF, 0xBB, 0xBF}, []byte("producto;precio\r\nCable 12 AWG;\"3.500,50\"\r\n")...)

		rows, style := readAll(t, domain.FormatCSV, data)
		require.Equal(t, [][]string{{"producto", "precio"}, {"Cable 12 AWG", "3.500,50"}}, rows)
		require.Equal(t, domain.DecimalComma, style)
	})

	t.Run("un CSV separado por comas usa punto decimal", func(t *testing.T) {
		rows, style := readAll(t, domain.FormatCSV, []byte("producto,precio\nCamiseta,25000.5\n"))
		require.Equal(t, [][]string{{"producto", "precio"}, {"Camiseta", "25000.5"}}, rows)
		require.Equal(t, domain.PlainNumbers, style)
	})

	t.Run("un CSV en Latin-1 se lee con sus tildes", func(t *testing.T) {
		latin1 := []byte("producto;categoria\nAcetaminof\xe9n;Analg\xe9sicos\n")

		rows, _ := readAll(t, domain.FormatCSV, latin1)
		require.Equal(t, []string{"Acetaminofén", "Analgésicos"}, rows[1])
	})

	t.Run("la plantilla CSV tiene el encabezado de la empresa", func(t *testing.T) {
		data, err := sheet.Sheets{}.Template(domain.FormatCSV, clothing)
		require.NoError(t, err)

		rows, _ := readAll(t, domain.FormatCSV, data)
		require.Equal(t, [][]string{clothing.Headers()}, rows)
	})
}

func TestXLSXRows(t *testing.T) {
	t.Run("la plantilla .xlsx se lee de vuelta con su encabezado y trae la ayuda", func(t *testing.T) {
		data, err := sheet.Sheets{}.Template(domain.FormatXLSX, clothing)
		require.NoError(t, err)

		rows, style := readAll(t, domain.FormatXLSX, data)
		require.Equal(t, [][]string{clothing.Headers()}, rows)
		require.Equal(t, domain.PlainNumbers, style)

		f, err := excelize.OpenReader(bytes.NewReader(data))
		require.NoError(t, err)
		help, err := f.GetRows("Ayuda")
		require.NoError(t, err)
		require.Contains(t, help, []string{"Unidades: und, par"})
		require.Contains(t, help, []string{"atributo: Material: Algodón, Cuero"})
		require.Contains(t, help, []string{"eje: talla: S, M"})
	})

	t.Run("lee los números como se guardan, sin el formato de la celda", func(t *testing.T) {
		f := excelize.NewFile()
		require.NoError(t, f.SetSheetRow("Sheet1", "A1", &[]any{"producto", "precio"}))
		require.NoError(t, f.SetSheetRow("Sheet1", "A2", &[]any{"Tubo PVC", 12500.5}))
		style, err := f.NewStyle(&excelize.Style{NumFmt: 4}) // #,##0.00
		require.NoError(t, err)
		require.NoError(t, f.SetCellStyle("Sheet1", "B2", "B2", style))
		buf, err := f.WriteToBuffer()
		require.NoError(t, err)

		rows, _ := readAll(t, domain.FormatXLSX, buf.Bytes())
		require.Equal(t, []string{"Tubo PVC", "12500.5"}, rows[1])
	})

	t.Run("rechaza un archivo que no es un libro de cálculo", func(t *testing.T) {
		_, err := sheet.Sheets{}.Rows(domain.FormatXLSX, []byte("producto;precio"))
		require.ErrorIs(t, err, domain.ErrUnreadableFile)
	})
}
