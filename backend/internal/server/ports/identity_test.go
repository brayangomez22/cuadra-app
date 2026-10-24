package ports

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	catalogapp "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	catalogdomain "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/templates"
)

func TestCatalogSeed(t *testing.T) {
	t.Run("la siembra lleva los datos de la plantilla, con sus atributos obligatorios", func(t *testing.T) {
		tests := []struct {
			name string
			tpl  templates.Template
			want catalogapp.Seed
		}{
			{
				name: "droguería: un atributo obligatorio y otro de sí/no",
				tpl: templates.Template{
					Units:      []templates.Unit{{Code: "und", Name: "Unidad"}, {Code: "caja", Name: "Caja"}},
					Categories: []string{"Medicamentos"},
					Attributes: []templates.Attribute{
						{Key: "laboratorio", Label: "Laboratorio", Type: "text", Required: true},
						{Key: "requiere_formula", Label: "Requiere fórmula", Type: "boolean"},
					},
				},
				want: catalogapp.Seed{
					Units:      []catalogapp.UnitSeed{{Code: "und", Name: "Unidad"}, {Code: "caja", Name: "Caja"}},
					Categories: []string{"Medicamentos"},
					Attributes: []catalogdomain.AttributeParams{
						{Key: "laboratorio", Label: "Laboratorio", Type: catalogdomain.AttributeText, Required: true},
						{Key: "requiere_formula", Label: "Requiere fórmula", Type: catalogdomain.AttributeBoolean},
					},
				},
			},
			{
				name: "ropa: atributo de lista y ejes de variante",
				tpl: templates.Template{
					DefaultTaxRate: decimal.RequireFromString("0.19"),
					Units:          []templates.Unit{{Code: "par", Name: "Par"}},
					Attributes:     []templates.Attribute{{Key: "genero", Label: "Género", Type: "list", Options: []string{"Mujer", "Hombre"}, Required: true}},
					VariantAxes:    []templates.Axis{{Name: "talla", Values: []string{"S", "M"}}},
				},
				want: catalogapp.Seed{
					Units: []catalogapp.UnitSeed{{Code: "par", Name: "Par"}},
					Attributes: []catalogdomain.AttributeParams{
						{Key: "genero", Label: "Género", Type: catalogdomain.AttributeList, Options: []string{"Mujer", "Hombre"}, Required: true},
					},
					VariantAxes: []catalogapp.AxisSeed{{Name: "talla", Values: []string{"S", "M"}}},
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				require.Equal(t, tt.want, catalogSeed(tt.tpl))
			})
		}
	})
}
