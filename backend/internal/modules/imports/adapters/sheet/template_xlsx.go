package sheet

import (
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// The template's sheets: the products are read from the first one.
const (
	productsSheet = "Productos"
	helpSheet     = "Ayuda"
)

// helpIntro explains how to fill the template.
var helpIntro = []string{
	"Cómo llenar la plantilla",
	"Una fila por variante. Las filas de un producto van seguidas y sus datos (descripción, categoría, unidad, IVA y atributos) se leen de la primera.",
	"Obligatorias: producto, unidad (su código), sku y precio. El precio no incluye IVA.",
	"IVA: 19, 19 % o 0,19. Si lo deja vacío, se usa el de la empresa.",
	"Categoría: si no existe, se crea.",
	"stock_inicial: solo entra a los productos nuevos, en la sede que elija al subir el archivo.",
	"Sí/no: escriba sí o no.",
}

// xlsxTemplate writes a workbook with the header row in the products sheet
// and a help sheet with the valid values of each column.
func xlsxTemplate(l domain.Layout) (_ []byte, err error) {
	f := excelize.NewFile()
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	if err := f.SetSheetName(f.GetSheetName(0), productsSheet); err != nil {
		return nil, err
	}
	if err := writeHeader(f, l.Headers()); err != nil {
		return nil, err
	}
	if _, err := f.NewSheet(helpSheet); err != nil {
		return nil, err
	}
	if err := writeHelp(f, helpLines(l)); err != nil {
		return nil, err
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeHeader(f *excelize.File, headers []string) error {
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}
	cells := make([]any, len(headers))
	for i, h := range headers {
		cells[i] = h
	}
	if err := f.SetSheetRow(productsSheet, "A1", &cells); err != nil {
		return err
	}
	last, err := excelize.CoordinatesToCellName(len(headers), 1)
	if err != nil {
		return err
	}
	if err := f.SetCellStyle(productsSheet, "A1", last, bold); err != nil {
		return err
	}
	end, err := excelize.ColumnNumberToName(len(headers))
	if err != nil {
		return err
	}
	return f.SetColWidth(productsSheet, "A", end, 18)
}

func writeHelp(f *excelize.File, lines []string) error {
	for i, line := range lines {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			return err
		}
		if err := f.SetCellStr(helpSheet, cell, line); err != nil {
			return err
		}
	}
	return f.SetColWidth(helpSheet, "A", "A", 120)
}

// helpLines are the help sheet's lines: the instructions and the values
// each column of the tenant accepts.
func helpLines(l domain.Layout) []string {
	lines := append([]string{}, helpIntro...)
	lines = append(lines, "", "Unidades: "+strings.Join(l.Units, ", "))
	for _, a := range l.Attributes {
		line := domain.AttributePrefix + a.Label
		if a.Required {
			line += " (obligatorio)"
		}
		if len(a.Options) > 0 {
			line += ": " + strings.Join(a.Options, ", ")
		}
		lines = append(lines, line)
	}
	if l.Variants {
		for _, a := range l.Axes {
			lines = append(lines, domain.AxisPrefix+a.Name+": "+strings.Join(a.Values, ", "))
		}
	}
	return lines
}
