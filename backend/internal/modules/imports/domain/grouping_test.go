package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// row is a valid row of a product with one variant.
func row(number int, product, sku string, axes map[string]string) domain.Row {
	return domain.Row{
		Number:  number,
		Product: domain.ProductFields{Name: product, Unit: "und", TaxRate: dec("0.19")},
		Variant: domain.VariantFields{SKU: sku, Price: dec("1000"), Axes: axes},
	}
}

// groupAll adds rows to a grouper and returns the groups it completes and
// the errors it reports.
func groupAll(t *testing.T, variants bool, rows ...domain.Row) ([]domain.Group, []domain.RowError) {
	t.Helper()
	g := domain.NewGrouper(variants)
	var groups []domain.Group
	var errs []domain.RowError
	for _, r := range rows {
		done, ok, rowErrs := g.Add(r, true)
		errs = append(errs, rowErrs...)
		if ok {
			groups = append(groups, done)
		}
	}
	if last, ok := g.Flush(); ok {
		groups = append(groups, last)
	}
	return groups, errs
}

func TestGrouper(t *testing.T) {
	t.Run("agrupa filas seguidas del mismo producto en un producto con sus variantes", func(t *testing.T) {
		tests := []struct {
			business string
			rows     []domain.Row
			want     []int // variants per product
		}{
			{"almacén de ropa: talla y color", []domain.Row{
				row(2, "Camiseta básica", "CAM-S", map[string]string{"talla": "S"}),
				row(3, " camiseta  BÁSICA", "CAM-M", map[string]string{"talla": "M"}),
				row(4, "Jean clásico", "JEAN-30", map[string]string{"talla": "30"}),
			}, []int{2, 1}},
			{"papelería: presentaciones", []domain.Row{
				row(2, "Cuaderno argollado", "CUA-100", map[string]string{"hojas": "100"}),
				row(3, "Cuaderno argollado", "CUA-200", map[string]string{"hojas": "200"}),
				row(4, "Cuaderno argollado", "CUA-300", map[string]string{"hojas": "300"}),
			}, []int{3}},
		}
		for _, tt := range tests {
			t.Run(tt.business, func(t *testing.T) {
				groups, errs := groupAll(t, true, tt.rows...)
				require.Empty(t, errs)
				require.Len(t, groups, len(tt.want))
				for i, n := range tt.want {
					require.Len(t, groups[i].Variants, n)
				}
				require.Equal(t, 2, groups[0].Row)
				require.Equal(t, 3, groups[0].Variants[1].Row)
			})
		}
	})

	t.Run("los datos del producto salen de su primera fila", func(t *testing.T) {
		second := row(3, "Camiseta básica", "CAM-M", nil)
		second.Product.Category = "Otra"
		first := row(2, "Camiseta básica", "CAM-S", nil)
		first.Product.Category = "Camisetas"

		groups, _ := groupAll(t, true, first, second)
		require.Equal(t, "Camisetas", groups[0].Product.Category)
	})

	t.Run("rechaza un producto cuyas filas no están seguidas", func(t *testing.T) {
		_, errs := groupAll(t, true,
			row(2, "Camiseta básica", "CAM-S", nil),
			row(3, "Jean clásico", "JEAN-30", nil),
			row(4, "Camiseta básica", "CAM-M", nil))
		require.Equal(t, []domain.RowError{{Row: 4, Column: "producto", Code: domain.CodeProductSplit}}, errs)
	})

	t.Run("rechaza un SKU repetido en el archivo, sin importar mayúsculas", func(t *testing.T) {
		_, errs := groupAll(t, true,
			row(2, "Tubo PVC", "TUB-1", nil),
			row(3, "Tubo CPVC", "tub-1 ", nil))
		require.Equal(t, []domain.RowError{{Row: 3, Column: "sku", Code: domain.CodeDuplicateSKU}}, errs)
	})

	t.Run("sin el módulo de variantes un producto en varias filas es un error", func(t *testing.T) {
		groups, errs := groupAll(t, false,
			row(2, "Tubo PVC", "TUB-1", nil),
			row(3, "Tubo PVC", "TUB-2", nil))
		require.Equal(t, []domain.RowError{{Row: 3, Column: "producto", Code: domain.CodeVariantsInactive}}, errs)
		require.Len(t, groups, 1)
	})

	t.Run("una fila inválida marca su producto como inválido", func(t *testing.T) {
		g := domain.NewGrouper(true)
		_, _, _ = g.Add(row(2, "Camiseta", "CAM-S", nil), true)
		_, _, _ = g.Add(row(3, "Camiseta", "CAM-M", nil), false)
		last, ok := g.Flush()
		require.True(t, ok)
		require.True(t, last.Invalid)
	})

	t.Run("los datos obligatorios del producto solo se exigen en su primera fila", func(t *testing.T) {
		missing := []domain.RowError{{Column: domain.ColumnUnit, Code: domain.CodeRequired}}
		first := row(2, "Camiseta", "CAM-S", nil)
		second := row(3, "Camiseta", "CAM-M", nil)
		second.ProductErrors = missing
		other := row(4, "Jean", "JEAN-30", nil)
		other.ProductErrors = []domain.RowError{{Row: 4, Column: domain.ColumnUnit, Code: domain.CodeRequired}}

		groups, errs := groupAll(t, true, first, second, other)
		require.Equal(t, other.ProductErrors, errs)
		require.False(t, groups[0].Invalid)
		require.True(t, groups[1].Invalid)
	})

	t.Run("las filas vacías no cortan un producto", func(t *testing.T) {
		groups, errs := groupAll(t, true,
			row(2, "Camiseta", "CAM-S", nil),
			domain.Row{Number: 3, Blank: true},
			row(4, "Camiseta", "CAM-M", nil))
		require.Empty(t, errs)
		require.Len(t, groups, 1)
		require.Len(t, groups[0].Variants, 2)
	})
}
