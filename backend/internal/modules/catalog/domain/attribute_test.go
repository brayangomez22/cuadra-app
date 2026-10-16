package domain_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func mustAttribute(t *testing.T, tenantID uuid.UUID, p domain.AttributeParams) *domain.AttributeDefinition {
	t.Helper()
	d, err := domain.NewAttributeDefinition(tenantID, p, testNow)
	require.NoError(t, err)
	return d
}

func TestNewAttributeDefinition(t *testing.T) {
	t.Run("crea una definición con la etiqueta recortada y las opciones en su orden", func(t *testing.T) {
		tests := []struct {
			name     string
			params   domain.AttributeParams
			wantOpts []string
		}{
			{"ferretería: material de lista", domain.AttributeParams{
				Key: "material", Label: " Material ", Type: domain.AttributeList, Options: []string{" PVC ", "Cobre", "Galvanizado"},
			}, []string{"PVC", "Cobre", "Galvanizado"}},
			{"almacén de ropa: marca de texto obligatoria", domain.AttributeParams{
				Key: "marca", Label: "Marca", Type: domain.AttributeText, Required: true,
			}, nil},
			{"droguería: requiere fórmula, sí o no", domain.AttributeParams{
				Key: "requiere_formula", Label: "Requiere fórmula", Type: domain.AttributeBoolean,
			}, nil},
			{"papelería: hojas, número", domain.AttributeParams{
				Key: "hojas", Label: "Hojas", Type: domain.AttributeNumber,
			}, nil},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tenantID := uuid.Must(uuid.NewV7())

				d, err := domain.NewAttributeDefinition(tenantID, tt.params, testNow)
				require.NoError(t, err)

				require.Equal(t, 7, int(d.ID().Version()))
				require.Equal(t, tenantID, d.TenantID())
				require.Equal(t, tt.params.Key, d.Key())
				require.Equal(t, strings.TrimSpace(tt.params.Label), d.Label())
				require.Equal(t, tt.params.Type, d.Type())
				require.Equal(t, tt.wantOpts, d.Options())
				require.Equal(t, tt.params.Required, d.Required())
				require.Equal(t, testNow.UTC(), d.CreatedAt())
			})
		}
	})

	t.Run("rechaza una clave con espacios, mayúsculas, tildes o de más de 50 caracteres", func(t *testing.T) {
		for _, key := range []string{"", "principio activo", "Marca", "tamaño", "1marca", "_marca", strings.Repeat("a", 51)} {
			_, err := domain.NewAttributeDefinition(uuid.Must(uuid.NewV7()), domain.AttributeParams{
				Key: key, Label: "Marca", Type: domain.AttributeText,
			}, testNow)
			require.ErrorIs(t, err, domain.ErrInvalidAttributeKey, "%q", key)
		}
	})

	t.Run("acepta una clave con números y guiones bajos", func(t *testing.T) {
		_, err := domain.NewAttributeDefinition(uuid.Must(uuid.NewV7()), domain.AttributeParams{
			Key: "principio_activo_2", Label: "Principio activo", Type: domain.AttributeText,
		}, testNow)
		require.NoError(t, err)
	})

	t.Run("rechaza una etiqueta vacía o de más de 100 caracteres", func(t *testing.T) {
		for _, label := range []string{"", "  ", strings.Repeat("a", 101)} {
			_, err := domain.NewAttributeDefinition(uuid.Must(uuid.NewV7()), domain.AttributeParams{
				Key: "marca", Label: label, Type: domain.AttributeText,
			}, testNow)
			require.ErrorIs(t, err, domain.ErrInvalidAttributeLabel, "%q", label)
		}
	})

	t.Run("rechaza un tipo que no existe", func(t *testing.T) {
		_, err := domain.ParseAttributeType("fecha")
		require.ErrorIs(t, err, domain.ErrInvalidAttributeType)

		_, err = domain.NewAttributeDefinition(uuid.Must(uuid.NewV7()), domain.AttributeParams{
			Key: "marca", Label: "Marca", Type: domain.AttributeType("fecha"),
		}, testNow)
		require.ErrorIs(t, err, domain.ErrInvalidAttributeType)
	})

	t.Run("una lista necesita opciones y los demás tipos no las aceptan", func(t *testing.T) {
		tests := []struct {
			name    string
			typ     domain.AttributeType
			options []string
		}{
			{"lista sin opciones", domain.AttributeList, nil},
			{"texto con opciones", domain.AttributeText, []string{"Acme"}},
			{"número con opciones", domain.AttributeNumber, []string{"1"}},
			{"sí/no con opciones", domain.AttributeBoolean, []string{"sí"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewAttributeDefinition(uuid.Must(uuid.NewV7()), domain.AttributeParams{
					Key: "material", Label: "Material", Type: tt.typ, Options: tt.options,
				}, testNow)
				require.ErrorIs(t, err, domain.ErrInvalidAttributeOptions)
			})
		}
	})

	t.Run("rechaza opciones vacías, largas o repetidas sin importar mayúsculas", func(t *testing.T) {
		for _, options := range [][]string{
			{"PVC", " "},
			{"PVC", strings.Repeat("a", 101)},
			{"PVC", "pvc"},
			{"Algodón", " algodón "},
		} {
			_, err := domain.NewAttributeDefinition(uuid.Must(uuid.NewV7()), domain.AttributeParams{
				Key: "material", Label: "Material", Type: domain.AttributeList, Options: options,
			}, testNow)
			require.ErrorIs(t, err, domain.ErrInvalidAttributeOptions, "%q", options)
		}
	})

	t.Run("rechaza un tenant vacío", func(t *testing.T) {
		_, err := domain.NewAttributeDefinition(uuid.Nil, domain.AttributeParams{
			Key: "marca", Label: "Marca", Type: domain.AttributeText,
		}, testNow)
		require.ErrorIs(t, err, domain.ErrInvalidTenantID)
	})
}

func TestAttributeDefinitionUpdate(t *testing.T) {
	t.Run("cambia la etiqueta, las opciones y si es obligatorio, y dice qué opciones quitó", func(t *testing.T) {
		d := mustAttribute(t, uuid.Must(uuid.NewV7()), domain.AttributeParams{
			Key: "material", Label: "Material", Type: domain.AttributeList, Options: []string{"PVC", "Cobre", "Hierro"},
		})

		removed, err := d.Update(" Material principal ", []string{"PVC", "Hierro", "Acero"}, true)
		require.NoError(t, err)

		require.Equal(t, []string{"Cobre"}, removed)
		require.Equal(t, "Material principal", d.Label())
		require.Equal(t, []string{"PVC", "Hierro", "Acero"}, d.Options())
		require.True(t, d.Required())
		require.Equal(t, "material", d.Key())
		require.Equal(t, domain.AttributeList, d.Type())
	})

	t.Run("cambiar una opción de mayúsculas cuenta como quitarla", func(t *testing.T) {
		d := mustAttribute(t, uuid.Must(uuid.NewV7()), domain.AttributeParams{
			Key: "color", Label: "Color", Type: domain.AttributeList, Options: []string{"rojo"},
		})

		removed, err := d.Update("Color", []string{"Rojo"}, false)
		require.NoError(t, err)
		require.Equal(t, []string{"rojo"}, removed)
	})

	t.Run("si la edición falla, la definición queda igual", func(t *testing.T) {
		d := mustAttribute(t, uuid.Must(uuid.NewV7()), domain.AttributeParams{
			Key: "material", Label: "Material", Type: domain.AttributeList, Options: []string{"PVC"},
		})

		_, err := d.Update("Material", nil, true)
		require.ErrorIs(t, err, domain.ErrInvalidAttributeOptions)
		_, err = d.Update("", []string{"Cobre"}, true)
		require.ErrorIs(t, err, domain.ErrInvalidAttributeLabel)

		require.Equal(t, "Material", d.Label())
		require.Equal(t, []string{"PVC"}, d.Options())
		require.False(t, d.Required())
	})
}

// catalogAttributes are the definitions of a tenant with attributes of every type.
func catalogAttributes(t *testing.T) []*domain.AttributeDefinition {
	t.Helper()
	tenantID := uuid.Must(uuid.NewV7())
	return []*domain.AttributeDefinition{
		mustAttribute(t, tenantID, domain.AttributeParams{Key: "marca", Label: "Marca", Type: domain.AttributeText, Required: true}),
		mustAttribute(t, tenantID, domain.AttributeParams{Key: "hojas", Label: "Hojas", Type: domain.AttributeNumber}),
		mustAttribute(t, tenantID, domain.AttributeParams{
			Key: "material", Label: "Material", Type: domain.AttributeList, Options: []string{"PVC", "Algodón"},
		}),
		mustAttribute(t, tenantID, domain.AttributeParams{Key: "importado", Label: "Importado", Type: domain.AttributeBoolean}),
	}
}

func text(s string) domain.AttributeValue { return domain.TextValue(s) }

func TestResolveAttributes(t *testing.T) {
	t.Run("acepta valores de cada tipo y los normaliza", func(t *testing.T) {
		tests := []struct {
			name string
			in   map[string]domain.AttributeValue
			want map[string]domain.AttributeValue
		}{
			{"ferretería: tubo de PVC importado",
				map[string]domain.AttributeValue{
					"marca": text(" Acme "), "material": text("pvc"), "importado": domain.BoolValue(true),
				},
				map[string]domain.AttributeValue{
					"marca": text("Acme"), "material": text("PVC"), "importado": domain.BoolValue(true),
				}},
			{"almacén de ropa: camiseta de algodón",
				map[string]domain.AttributeValue{"marca": text("Textiles"), "material": text(" ALGODÓN ")},
				map[string]domain.AttributeValue{"marca": text("Textiles"), "material": text("Algodón")}},
			{"papelería: cuaderno de 100 hojas",
				map[string]domain.AttributeValue{"marca": text("Norma"), "hojas": text(" 100.00 "), "importado": domain.BoolValue(false)},
				map[string]domain.AttributeValue{"marca": text("Norma"), "hojas": text("100"), "importado": domain.BoolValue(false)}},
			{"un número decimal",
				map[string]domain.AttributeValue{"marca": text("Acme"), "hojas": text("2.50")},
				map[string]domain.AttributeValue{"marca": text("Acme"), "hojas": text("2.5")}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				attrs, err := domain.ResolveAttributes(catalogAttributes(t), tt.in)
				require.NoError(t, err)
				require.Equal(t, tt.want, attrs.Values())
			})
		}
	})

	t.Run("un atributo de tipo número rechaza texto", func(t *testing.T) {
		for _, value := range []string{"cien", "1,5", "12abc", strings.Repeat("1", 41)} {
			_, err := domain.ResolveAttributes(catalogAttributes(t), map[string]domain.AttributeValue{
				"marca": text("Acme"), "hojas": text(value),
			})
			require.ErrorIs(t, err, domain.ErrInvalidAttributeValue, "%q", value)
			var attrErr *domain.AttributeError
			require.ErrorAs(t, err, &attrErr)
			require.Equal(t, "hojas", attrErr.Key)
			require.Equal(t, "Hojas", attrErr.Label)
		}
	})

	t.Run("un atributo de lista rechaza una opción no definida", func(t *testing.T) {
		_, err := domain.ResolveAttributes(catalogAttributes(t), map[string]domain.AttributeValue{
			"marca": text("Acme"), "material": text("Cobre"),
		})
		require.ErrorIs(t, err, domain.ErrInvalidAttributeValue)
		var attrErr *domain.AttributeError
		require.ErrorAs(t, err, &attrErr)
		require.Equal(t, "material", attrErr.Key)
	})

	t.Run("un atributo obligatorio faltante falla", func(t *testing.T) {
		_, err := domain.ResolveAttributes(catalogAttributes(t), map[string]domain.AttributeValue{"hojas": text("100")})
		require.ErrorIs(t, err, domain.ErrAttributeRequired)
		var attrErr *domain.AttributeError
		require.ErrorAs(t, err, &attrErr)
		require.Equal(t, "marca", attrErr.Key)
		require.Equal(t, "Marca", attrErr.Label)
	})

	t.Run("un texto vacío cuenta como faltante", func(t *testing.T) {
		_, err := domain.ResolveAttributes(catalogAttributes(t), map[string]domain.AttributeValue{"marca": text("   ")})
		require.ErrorIs(t, err, domain.ErrAttributeRequired)

		attrs, err := domain.ResolveAttributes(catalogAttributes(t), map[string]domain.AttributeValue{
			"marca": text("Acme"), "hojas": text(" "),
		})
		require.NoError(t, err)
		require.Equal(t, map[string]domain.AttributeValue{"marca": text("Acme")}, attrs.Values())
	})

	t.Run("rechaza un atributo que el tenant no definió", func(t *testing.T) {
		_, err := domain.ResolveAttributes(catalogAttributes(t), map[string]domain.AttributeValue{
			"marca": text("Acme"), "talla": text("M"),
		})
		require.ErrorIs(t, err, domain.ErrUnknownAttribute)
		var attrErr *domain.AttributeError
		require.ErrorAs(t, err, &attrErr)
		require.Equal(t, "talla", attrErr.Key)
	})

	t.Run("un atributo de sí/no rechaza un texto y uno de texto rechaza un booleano", func(t *testing.T) {
		_, err := domain.ResolveAttributes(catalogAttributes(t), map[string]domain.AttributeValue{
			"marca": text("Acme"), "importado": text("sí"),
		})
		require.ErrorIs(t, err, domain.ErrInvalidAttributeValue)

		_, err = domain.ResolveAttributes(catalogAttributes(t), map[string]domain.AttributeValue{"marca": domain.BoolValue(true)})
		require.ErrorIs(t, err, domain.ErrInvalidAttributeValue)
	})

	t.Run("rechaza un texto de más de 500 caracteres", func(t *testing.T) {
		_, err := domain.ResolveAttributes(catalogAttributes(t), map[string]domain.AttributeValue{"marca": text(strings.Repeat("a", 501))})
		require.ErrorIs(t, err, domain.ErrInvalidAttributeValue)
	})

	t.Run("sin definiciones solo se aceptan atributos vacíos", func(t *testing.T) {
		attrs, err := domain.ResolveAttributes(nil, nil)
		require.NoError(t, err)
		require.Empty(t, attrs.Values())

		_, err = domain.ResolveAttributes(nil, map[string]domain.AttributeValue{"marca": text("Acme")})
		require.ErrorIs(t, err, domain.ErrUnknownAttribute)
	})
}

func TestRehydrateAttributes(t *testing.T) {
	t.Run("reconstruye los valores guardados sin compararlos con las definiciones", func(t *testing.T) {
		values := map[string]domain.AttributeValue{"marca": text("Acme"), "importado": domain.BoolValue(false)}

		attrs, err := domain.RehydrateAttributes(values)
		require.NoError(t, err)
		require.Equal(t, values, attrs.Values())
	})

	t.Run("rechaza una clave o un texto que no se pudieron guardar", func(t *testing.T) {
		for _, values := range []map[string]domain.AttributeValue{
			{"Marca": text("Acme")},
			{"marca": text("")},
		} {
			_, err := domain.RehydrateAttributes(values)
			require.Error(t, err, "%v", values)
		}
	})

	t.Run("los valores devueltos son una copia", func(t *testing.T) {
		attrs, err := domain.RehydrateAttributes(map[string]domain.AttributeValue{"marca": text("Acme")})
		require.NoError(t, err)

		attrs.Values()["marca"] = text("Otra")
		require.Equal(t, text("Acme"), attrs.Values()["marca"])
	})
}
