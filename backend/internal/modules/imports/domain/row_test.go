package domain_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// hardwareLayout is the layout of a hardware store: a required brand and
// no variants.
func hardwareLayout() domain.Layout {
	return domain.Layout{
		DefaultTaxRate: dec("0.19"),
		Attributes: []domain.AttributeColumn{
			{Key: "marca", Label: "Marca", Type: domain.KindText, Required: true},
			{Key: "peso", Label: "Peso", Type: domain.KindNumber},
		},
	}
}

// clothingLayout is the layout of a clothing store with the variants
// module on: size and color axes and a yes/no attribute.
func clothingLayout() domain.Layout {
	return domain.Layout{
		DefaultTaxRate: dec("0.19"),
		Variants:       true,
		Attributes:     []domain.AttributeColumn{{Key: "importado", Label: "Importado", Type: domain.KindBoolean}},
		Axes:           []domain.AxisColumn{{Name: "talla", Values: []string{"S", "M"}}, {Name: "color", Values: []string{"Rojo"}}},
	}
}

func header(t *testing.T, l domain.Layout, cells ...string) domain.Header {
	t.Helper()
	h, errs := domain.ParseHeader(l, cells)
	require.Empty(t, errs)
	return h
}

func TestParseHeader(t *testing.T) {
	t.Run("lee las columnas en cualquier orden, sin importar mayúsculas, espacios ni tildes", func(t *testing.T) {
		h := header(t, hardwareLayout(), " SKU", "Precio", "Producto", "Unidad", "Descripción", "atributo: MARCA")
		row, errs := h.Parse(2, []string{"tub-1", "10000", "Tubo PVC", "m", "Tramo de 6 m", "Pavco"}, domain.PlainNumbers)
		require.Empty(t, errs)
		require.Equal(t, "Tubo PVC", row.Product.Name)
		require.Equal(t, "Tramo de 6 m", row.Product.Description)
		require.Equal(t, domain.AttributeValue{Text: "Pavco"}, row.Product.Attributes["marca"])
	})

	t.Run("reporta en la fila 1 las columnas obligatorias que faltan", func(t *testing.T) {
		_, errs := domain.ParseHeader(hardwareLayout(), []string{"producto", "sku"})
		require.ElementsMatch(t, []domain.RowError{
			{Row: 1, Column: "unidad", Code: domain.CodeMissingColumn},
			{Row: 1, Column: "precio", Code: domain.CodeMissingColumn},
		}, errs)
	})

	t.Run("reporta una columna desconocida o repetida", func(t *testing.T) {
		_, errs := domain.ParseHeader(hardwareLayout(), []string{"producto", "unidad", "sku", "precio", "color", "sku"})
		require.ElementsMatch(t, []domain.RowError{
			{Row: 1, Column: "color", Code: domain.CodeUnknownColumn},
			{Row: 1, Column: "sku", Code: domain.CodeDuplicateColumn},
		}, errs)
	})

	t.Run("sin el módulo de variantes una columna de eje es desconocida", func(t *testing.T) {
		l := clothingLayout()
		l.Variants = false
		_, errs := domain.ParseHeader(l, []string{"producto", "unidad", "sku", "precio", "eje: talla"})
		require.Equal(t, []domain.RowError{{Row: 1, Column: "eje: talla", Code: domain.CodeUnknownColumn}}, errs)
	})

	t.Run("los encabezados de la plantilla se leen sin errores", func(t *testing.T) {
		for _, l := range []domain.Layout{hardwareLayout(), clothingLayout()} {
			_, errs := domain.ParseHeader(l, l.Headers())
			require.Empty(t, errs)
		}
	})
}

func TestHeaderParse(t *testing.T) {
	clothing := header(t, clothingLayout(), clothingLayout().Headers()...)

	t.Run("lee una fila completa", func(t *testing.T) {
		tests := []struct {
			business string
			h        domain.Header
			cells    map[string]string
			check    func(t *testing.T, r domain.Row)
		}{
			{"ferretería", header(t, hardwareLayout(), hardwareLayout().Headers()...), map[string]string{
				"producto": "Tubo PVC 1/2", "categoría": "Tubería", "unidad": "m", "iva": "19%", "sku": "TUB-12",
				"código_barras": "7701234567890", "costo": "8333.33", "precio": "10000", "stock_inicial": "120.5",
				"atributo: Marca": "Pavco", "atributo: Peso": "40",
			}, func(t *testing.T, r domain.Row) {
				t.Helper()
				require.Equal(t, "Tubería", r.Product.Category)
				require.True(t, dec("0.19").Equal(r.Product.TaxRate))
				require.True(t, dec("8333.33").Equal(r.Variant.Cost))
				require.True(t, r.HasStock)
				require.True(t, dec("120.5").Equal(r.Stock))
				require.Equal(t, domain.AttributeValue{Text: "40"}, r.Product.Attributes["peso"])
			}},
			{"almacén de ropa", clothing, map[string]string{
				"producto": "Camiseta", "unidad": "und", "sku": "CAM-S", "precio": "25000", "iva": "0.19",
				"eje: talla": "S", "eje: color": "Rojo", "atributo: Importado": "Sí",
			}, func(t *testing.T, r domain.Row) {
				t.Helper()
				require.Equal(t, map[string]string{"talla": "S", "color": "Rojo"}, r.Variant.Axes)
				require.Equal(t, domain.AttributeValue{Bool: true, IsBool: true}, r.Product.Attributes["importado"])
				require.False(t, r.HasStock)
				require.True(t, r.Variant.Cost.IsZero())
			}},
		}
		for _, tt := range tests {
			t.Run(tt.business, func(t *testing.T) {
				row, errs := tt.h.Parse(7, cellsOf(tt.h, tt.cells), domain.PlainNumbers)
				require.Empty(t, errs)
				require.Equal(t, 7, row.Number)
				tt.check(t, row)
			})
		}
	})

	t.Run("reporta cada error de la fila con su columna", func(t *testing.T) {
		h := header(t, hardwareLayout(), hardwareLayout().Headers()...)
		cells := cellsOf(h, map[string]string{
			"producto": "Tubo", "unidad": "m", "sku": "TUB-1", "precio": "diez mil", "costo": "-5",
			"stock_inicial": "abc", "atributo: Peso": "grueso",
		})

		row, errs := h.Parse(154, cells, domain.PlainNumbers)
		require.ElementsMatch(t, []domain.RowError{
			{Row: 154, Column: "precio", Code: domain.CodeNotANumber},
			{Row: 154, Column: "costo", Code: domain.CodeNegativeNumber},
			{Row: 154, Column: "stock_inicial", Code: domain.CodeNotANumber},
			{Row: 154, Column: "atributo: Peso", Code: domain.CodeNotANumber},
		}, errs)
		require.Equal(t, []domain.RowError{{Row: 154, Column: "atributo: Marca", Code: domain.CodeRequired}}, row.ProductErrors,
			"a required product value only counts in the product's first row")
	})

	t.Run("reporta los valores obligatorios vacíos y un sí/no inválido", func(t *testing.T) {
		cells := cellsOf(clothing, map[string]string{"producto": " ", "atributo: Importado": "tal vez"})

		row, errs := clothing.Parse(3, cells, domain.PlainNumbers)
		require.ElementsMatch(t, []domain.RowError{
			{Row: 3, Column: "producto", Code: domain.CodeRequired},
			{Row: 3, Column: "sku", Code: domain.CodeRequired},
			{Row: 3, Column: "precio", Code: domain.CodeRequired},
			{Row: 3, Column: "atributo: Importado", Code: domain.CodeInvalidBoolean},
		}, errs)
		require.Equal(t, []domain.RowError{{Row: 3, Column: "unidad", Code: domain.CodeRequired}}, row.ProductErrors)
	})

	t.Run("una fila vacía se marca vacía, sin errores", func(t *testing.T) {
		row, errs := clothing.Parse(9, []string{"", " ", ""}, domain.PlainNumbers)
		require.Empty(t, errs)
		require.True(t, row.Blank)
	})

	t.Run("sin IVA toma el de la empresa", func(t *testing.T) {
		row, errs := clothing.Parse(2, cellsOf(clothing, map[string]string{
			"producto": "Camiseta", "unidad": "und", "sku": "CAM-S", "precio": "25000",
		}), domain.PlainNumbers)
		require.Empty(t, errs)
		require.True(t, dec("0.19").Equal(row.Product.TaxRate))
	})
}

func TestParseNumber(t *testing.T) {
	tests := []struct {
		in    string
		style domain.NumberStyle
		want  string
	}{
		{"12500", domain.PlainNumbers, "12500"},
		{"12500.5", domain.PlainNumbers, "12500.5"},
		{"$ 12,500.50", domain.PlainNumbers, "12500.50"},
		{"12.500,50", domain.DecimalComma, "12500.50"},
		{"$12.500", domain.DecimalComma, "12500"},
		{"2,375", domain.DecimalComma, "2.375"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := domain.ParseNumber(tt.in, tt.style)
			require.True(t, ok)
			require.True(t, dec(tt.want).Equal(got), "got %s", got)
		})
	}
	t.Run("rechaza lo que no es un número", func(t *testing.T) {
		for _, in := range []string{"diez", "1.2.3,4,5", "12a", "$"} {
			_, ok := domain.ParseNumber(in, domain.DecimalComma)
			require.False(t, ok, in)
		}
	})
}

func TestParseTaxRate(t *testing.T) {
	for in, want := range map[string]string{"19%": "0.19", "19 %": "0.19", "19": "0.19", "0.19": "0.19", "0": "0", "5": "0.05"} {
		t.Run(in, func(t *testing.T) {
			got, ok := domain.ParseTaxRate(in, domain.PlainNumbers)
			require.True(t, ok)
			require.True(t, dec(want).Equal(got), "got %s", got)
		})
	}
}

// cellsOf places values (by column name) in the order of h's columns.
func cellsOf(h domain.Header, values map[string]string) []string {
	cells := make([]string, len(h.Columns()))
	for i, name := range h.Columns() {
		cells[i] = values[name]
	}
	return cells
}
