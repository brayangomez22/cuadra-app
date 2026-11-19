package sheet_test

import (
	"bytes"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/sheet"
)

func sample() sheet.Table {
	t := sheet.Table{Name: "Ventas", Columns: []string{"Producto", "Cantidad", "Ingreso"}}
	t.Add(sheet.Text("Cemento gris 50kg"), sheet.Number(decimal.RequireFromString("2.5")), sheet.Number(decimal.RequireFromString("75000.50")))
	t.Add(sheet.Text("Camiseta básica; roja"), sheet.Count(3), sheet.Number(decimal.RequireFromString("-1200")))
	return t
}

func TestWrite(t *testing.T) {
	t.Run("el CSV va con BOM, punto y coma y coma decimal", func(t *testing.T) {
		data, err := sheet.Write(sample(), sheet.CSV)
		require.NoError(t, err)
		require.True(t, bytes.HasPrefix(data, []byte("\xEF\xBB\xBF")))
		text := string(data[3:])
		require.Contains(t, text, "Producto;Cantidad;Ingreso\r\n")
		require.Contains(t, text, "Cemento gris 50kg;2,5;75000,5\r\n")
		require.Contains(t, text, `"Camiseta básica; roja";3;-1200`)
	})
	t.Run("el xlsx tiene la hoja del reporte con las cifras como números", func(t *testing.T) {
		data, err := sheet.Write(sample(), sheet.XLSX)
		require.NoError(t, err)
		f, err := excelize.OpenReader(bytes.NewReader(data))
		require.NoError(t, err)
		defer func() { _ = f.Close() }()
		rows, err := f.GetRows("Ventas")
		require.NoError(t, err)
		require.Equal(t, []string{"Producto", "Cantidad", "Ingreso"}, rows[0])
		require.Equal(t, "Cemento gris 50kg", rows[1][0])
		kind, err := f.GetCellType("Ventas", "C2")
		require.NoError(t, err)
		require.NotEqual(t, excelize.CellTypeSharedString, kind, "la cifra es un número, no un texto")
		v, err := f.GetCellValue("Ventas", "C2")
		require.NoError(t, err)
		require.Equal(t, "75000.5", v)
	})
	t.Run("rechaza un formato desconocido", func(t *testing.T) {
		_, err := sheet.Write(sample(), sheet.Format("pdf"))
		require.ErrorIs(t, err, sheet.ErrUnknownFormat)
	})
}
