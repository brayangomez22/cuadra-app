package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestSeedTenant(t *testing.T) {
	// Seeds of two kinds of business, as their templates declare them.
	hardware := app.Seed{
		Units:      []app.UnitSeed{{Code: "und", Name: "Unidad"}, {Code: "m", Name: "Metro"}, {Code: "bulto", Name: "Bulto"}},
		Categories: []string{"Herramientas", "Eléctricos"},
		Attributes: []domain.AttributeParams{{Key: "marca", Label: "Marca", Type: domain.AttributeText}},
	}
	clothing := app.Seed{
		Units:      []app.UnitSeed{{Code: "und", Name: "Unidad"}, {Code: "par", Name: "Par"}},
		Categories: []string{"Mujer", "Hombre", "Calzado"},
		Attributes: []domain.AttributeParams{
			{Key: "marca", Label: "Marca", Type: domain.AttributeText},
			{Key: "genero", Label: "Género", Type: domain.AttributeList, Options: []string{"Mujer", "Hombre", "Unisex"}},
		},
		VariantAxes: []app.AxisSeed{
			{Name: "talla", Values: []string{"S", "M", "L"}},
			{Name: "color", Values: []string{"Negro", "Blanco"}},
		},
	}

	tests := []struct {
		name       string
		seed       app.Seed
		units      map[string]string
		categories []string
		attributes []string
		axes       map[string][]string
	}{
		{
			name:       "ferretería",
			seed:       hardware,
			units:      map[string]string{"und": "Unidad", "m": "Metro", "bulto": "Bulto"},
			categories: []string{"Eléctricos", "Herramientas"},
			attributes: []string{"marca"},
			axes:       map[string][]string{},
		},
		{
			name:       "ropa y calzado",
			seed:       clothing,
			units:      map[string]string{"und": "Unidad", "par": "Par"},
			categories: []string{"Calzado", "Hombre", "Mujer"},
			attributes: []string{"genero", "marca"},
			axes:       map[string][]string{"talla": {"S", "M", "L"}, "color": {"Negro", "Blanco"}},
		},
	}
	for _, tt := range tests {
		t.Run("sembrar un tenant crea exactamente lo de su siembra: "+tt.name, func(t *testing.T) {
			e := newEnv(t)
			tenant := app.Actor{TenantID: uuid.Must(uuid.NewV7())}

			require.NoError(t, e.svc.SeedTenant(t.Context(), tenant.TenantID, tt.seed))

			units, err := e.svc.ListUnits(t.Context(), tenant)
			require.NoError(t, err)
			gotUnits := map[string]string{}
			for _, u := range units {
				gotUnits[u.Code()] = u.Name()
			}
			require.Equal(t, tt.units, gotUnits)

			categories, err := e.svc.ListCategories(t.Context(), tenant)
			require.NoError(t, err)
			gotCategories := []string{}
			for _, c := range categories {
				require.Nil(t, c.ParentID())
				gotCategories = append(gotCategories, c.Name())
			}
			require.Equal(t, tt.categories, gotCategories)

			attributes, err := e.svc.ListAttributes(t.Context(), tenant)
			require.NoError(t, err)
			gotAttributes := []string{}
			for _, d := range attributes {
				gotAttributes = append(gotAttributes, d.Key())
			}
			require.ElementsMatch(t, tt.attributes, gotAttributes)

			axes, err := e.svc.ListVariantAxes(t.Context(), tenant)
			require.NoError(t, err)
			gotAxes := map[string][]string{}
			for _, a := range axes {
				gotAxes[a.Name()] = a.Values()
			}
			require.Equal(t, tt.axes, gotAxes)

			_, attrs := e.span(t, "catalog.SeedTenant")
			require.Equal(t, attribute.StringValue(tenant.TenantID.String()), attrs["tenant.id"])
		})
	}

	t.Run("si un dato de la siembra es inválido, no guarda nada", func(t *testing.T) {
		tests := []struct {
			name    string
			mutate  func(*app.Seed)
			wantErr error
		}{
			{name: "unidad con código inválido", mutate: func(s *app.Seed) { s.Units[1].Code = "par de" }, wantErr: domain.ErrInvalidUnitCode},
			{name: "unidad repetida", mutate: func(s *app.Seed) { s.Units[1].Code = "und" }, wantErr: domain.ErrUnitCodeTaken},
			{name: "categoría sin nombre", mutate: func(s *app.Seed) { s.Categories[2] = " " }, wantErr: domain.ErrInvalidCategoryName},
			{name: "atributo de lista sin opciones", mutate: func(s *app.Seed) { s.Attributes[1].Options = nil }, wantErr: domain.ErrInvalidAttributeOptions},
			{name: "eje sin valores", mutate: func(s *app.Seed) { s.VariantAxes[1].Values = nil }, wantErr: domain.ErrInvalidAxisAllowedValues},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newEnv(t)
				tenant := app.Actor{TenantID: uuid.Must(uuid.NewV7())}
				seed := cloneSeed(clothing)
				tt.mutate(&seed)

				err := e.svc.SeedTenant(t.Context(), tenant.TenantID, seed)

				require.ErrorIs(t, err, tt.wantErr)
				units, err := e.svc.ListUnits(t.Context(), tenant)
				require.NoError(t, err)
				require.Empty(t, units)
				categories, err := e.svc.ListCategories(t.Context(), tenant)
				require.NoError(t, err)
				require.Empty(t, categories)
				attributes, err := e.svc.ListAttributes(t.Context(), tenant)
				require.NoError(t, err)
				require.Empty(t, attributes)
				axes, err := e.svc.ListVariantAxes(t.Context(), tenant)
				require.NoError(t, err)
				require.Empty(t, axes)
			})
		}
	})
}

// cloneSeed copies s deep enough for a test to change it.
func cloneSeed(s app.Seed) app.Seed {
	c := app.Seed{
		Units:      append([]app.UnitSeed(nil), s.Units...),
		Categories: append([]string(nil), s.Categories...),
	}
	for _, a := range s.Attributes {
		a.Options = append([]string(nil), a.Options...)
		c.Attributes = append(c.Attributes, a)
	}
	for _, a := range s.VariantAxes {
		a.Values = append([]string(nil), a.Values...)
		c.VariantAxes = append(c.VariantAxes, a)
	}
	return c
}
