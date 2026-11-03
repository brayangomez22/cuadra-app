package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

func TestHeaderParseReorder(t *testing.T) {
	tests := []struct {
		name     string
		l        domain.Layout
		min, max string
		want     *domain.Reorder
		errs     []domain.RowError
	}{
		{name: "ferretería: lee el mínimo y el máximo, fraccionarios", l: hardwareLayout(),
			min: "12.5", max: "60", want: &domain.Reorder{Min: dec("12.5"), Max: dec("60")}},
		{name: "almacén de ropa: un mínimo igual al máximo es válido", l: clothingLayout(),
			min: "4", max: "4", want: &domain.Reorder{Min: dec("4"), Max: dec("4")}},
		{name: "un mínimo de cero es válido", l: clothingLayout(),
			min: "0", max: "10", want: &domain.Reorder{Min: dec("0"), Max: dec("10")}},
		{name: "sin mínimo ni máximo no hay regla", l: hardwareLayout()},
		{name: "un mínimo sin máximo pide el máximo", l: hardwareLayout(), min: "5",
			errs: []domain.RowError{{Row: 4, Column: "stock_máximo", Code: domain.CodeReorderIncomplete}}},
		{name: "un máximo sin mínimo pide el mínimo", l: clothingLayout(), max: "5",
			errs: []domain.RowError{{Row: 4, Column: "stock_mínimo", Code: domain.CodeReorderIncomplete}}},
		{name: "un mínimo mayor que el máximo es un error", l: clothingLayout(), min: "20", max: "10",
			errs: []domain.RowError{{Row: 4, Column: "stock_mínimo", Code: domain.CodeReorderRange}}},
		{name: "un mínimo negativo es un error", l: hardwareLayout(), min: "-1", max: "10",
			errs: []domain.RowError{{Row: 4, Column: "stock_mínimo", Code: domain.CodeNegativeNumber}}},
		{name: "un máximo que no es número es un error", l: hardwareLayout(), min: "1", max: "muchos",
			errs: []domain.RowError{{Row: 4, Column: "stock_máximo", Code: domain.CodeNotANumber}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := header(t, tt.l, tt.l.Headers()...)
			cells := cellsOf(h, map[string]string{
				"producto": "Producto", "unidad": "und", "sku": "SKU-1", "precio": "1000",
				"atributo: Marca": "Genérica", "stock_mínimo": tt.min, "stock_máximo": tt.max,
			})

			row, errs := h.Parse(4, cells, domain.PlainNumbers)
			require.Equal(t, tt.errs, errs)
			if tt.want == nil {
				require.False(t, row.HasReorder)
				return
			}
			require.True(t, row.HasReorder)
			require.True(t, tt.want.Min.Equal(row.Reorder.Min), "min %s", row.Reorder.Min)
			require.True(t, tt.want.Max.Equal(row.Reorder.Max), "max %s", row.Reorder.Max)
		})
	}
}

func TestReorderColumns(t *testing.T) {
	t.Run("la plantilla trae las columnas de mínimo y máximo", func(t *testing.T) {
		require.Subset(t, hardwareLayout().Headers(), []string{"stock_mínimo", "stock_máximo"})
	})

	t.Run("se leen sin tildes ni mayúsculas, y son opcionales", func(t *testing.T) {
		h := header(t, clothingLayout(), "producto", "unidad", "sku", "precio", "Stock_Minimo", "STOCK_MAXIMO")
		row, errs := h.Parse(2, []string{"Camiseta", "und", "CAM-S", "25000", "2", "8"}, domain.PlainNumbers)
		require.Empty(t, errs)
		require.True(t, row.HasReorder)
	})

	t.Run("cada variante de un producto lleva su propia regla", func(t *testing.T) {
		h := header(t, clothingLayout(), "producto", "unidad", "sku", "precio", "stock_mínimo", "stock_máximo")
		g := domain.NewGrouper(true)
		for i, cells := range [][]string{
			{"Camiseta", "und", "CAM-S", "25000", "2", "8"},
			{"Camiseta", "und", "CAM-M", "25000", "", ""},
		} {
			row, errs := h.Parse(i+2, cells, domain.PlainNumbers)
			require.Empty(t, errs)
			g.Add(row, true)
		}

		group, ok := g.Flush()
		require.True(t, ok)
		require.True(t, group.Variants[0].HasReorder)
		require.True(t, dec("8").Equal(group.Variants[0].Reorder.Max))
		require.False(t, group.Variants[1].HasReorder)
	})
}
