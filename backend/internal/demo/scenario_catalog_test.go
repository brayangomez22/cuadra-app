package demo_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/demo"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/templates"
	business "github.com/brayangomez22/cuadra-app/backend/templates/business"
)

func variants(s demo.Scenario) []demo.Variant {
	var out []demo.Variant
	for _, p := range s.Products {
		out = append(out, p.Variants...)
	}
	return out
}

func eventTimes(s demo.Scenario) []time.Time {
	var out []time.Time
	for _, d := range s.Days {
		for _, e := range d.Events() {
			out = append(out, e.At)
		}
	}
	return out
}

// validEAN13 reports whether code is 13 digits with a valid check digit.
func validEAN13(code string) bool {
	if len(code) != 13 {
		return false
	}
	sum := 0
	for i, r := range code[:12] {
		if r < '0' || r > '9' {
			return false
		}
		w := 1
		if i%2 == 1 {
			w = 3
		}
		sum += int(r-'0') * w
	}
	return int(code[12]-'0') == (10-sum%10)%10
}

func TestScenarioCatalog(t *testing.T) {
	for _, template := range demoTemplates {
		t.Run("los SKU y los códigos de barras no se repiten y tienen dígito válido: "+template, func(t *testing.T) {
			skus, barcodes := map[string]bool{}, map[string]bool{}
			for _, v := range variants(small(t, template, 2)) {
				require.False(t, skus[v.SKU], v.SKU)
				require.False(t, barcodes[v.Barcode], v.Barcode)
				require.True(t, validEAN13(v.Barcode), v.Barcode)
				require.True(t, v.Price.GreaterThan(v.Cost), "%s sells above its cost", v.SKU)
				skus[v.SKU], barcodes[v.Barcode] = true, true
			}
		})

		t.Run("el perfil usa solo unidades, categorías, atributos y ejes de su plantilla: "+template, func(t *testing.T) {
			registry, err := modules.NewRegistry(modules.Builtin()...)
			require.NoError(t, err)
			catalog, err := templates.Load(business.FS, registry.CheckSet)
			require.NoError(t, err)
			tpl, ok := catalog.Get(template)
			require.True(t, ok)

			for _, p := range small(t, template, 2).Products {
				require.True(t, slices.ContainsFunc(tpl.Units, func(u templates.Unit) bool { return u.Code == p.Unit }), p.Unit)
				require.Contains(t, tpl.Categories, p.Category)
				for key := range p.Attributes {
					require.True(t, slices.ContainsFunc(tpl.Attributes, func(a templates.Attribute) bool { return a.Key == key }), key)
				}
				for _, v := range p.Variants {
					for axis, value := range v.Axes {
						j := slices.IndexFunc(tpl.VariantAxes, func(a templates.Axis) bool { return a.Name == axis })
						require.GreaterOrEqual(t, j, 0, axis)
						require.Contains(t, tpl.VariantAxes[j].Values, value)
					}
				}
			}
		})
	}

	t.Run("ropa reparte sus productos en variantes por talla y color", func(t *testing.T) {
		s := small(t, "ropa_calzado", 2)
		withMany := 0
		for _, p := range s.Products {
			if len(p.Variants) > 1 {
				withMany++
			}
			for _, v := range p.Variants {
				require.Contains(t, v.Axes, "color")
			}
		}
		require.Greater(t, withMany, len(s.Products)/2)
	})

	t.Run("la ferretería vende productos simples, sin ejes", func(t *testing.T) {
		for _, p := range small(t, "ferreteria", 2).Products {
			require.Len(t, p.Variants, 1)
			require.Empty(t, p.Variants[0].Axes)
		}
	})
}
