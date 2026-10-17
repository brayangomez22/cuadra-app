package templates_test

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/templates"
	"github.com/brayangomez22/cuadra-app/backend/templates/business"
)

// builtinModules checks a template's modules against Cuadra's registry, as
// cmd/api does at startup.
func builtinModules(t *testing.T) func([]string) error {
	t.Helper()
	registry, err := modules.NewRegistry(modules.Builtin()...)
	require.NoError(t, err)
	return registry.CheckSet
}

// TestBusinessTemplates validates the templates the API embeds: it is what
// checks every YAML against its schema in CI.
func TestBusinessTemplates(t *testing.T) {
	catalog, err := templates.Load(business.FS, builtinModules(t))
	require.NoError(t, err)

	t.Run("cada plantilla embebida cumple su esquema y carga sin errores, y están las seis iniciales", func(t *testing.T) {
		ids := []string{}
		for _, tpl := range catalog.List() {
			ids = append(ids, tpl.ID)
			require.NotEmpty(t, tpl.Name, tpl.ID)
			require.NotEmpty(t, tpl.Description, tpl.ID)
			require.NotEmpty(t, tpl.Units, "%s: a tenant without units cannot create products", tpl.ID)
		}
		require.ElementsMatch(t, []string{"generica", "ferreteria", "tienda", "drogueria", "papeleria", "ropa_calzado"}, ids)
	})

	t.Run("ferretería: conserva las unidades de antes y no trae variantes", func(t *testing.T) {
		tpl, ok := catalog.Get("ferreteria")
		require.True(t, ok)
		codes := []string{}
		for _, u := range tpl.Units {
			codes = append(codes, u.Code)
		}
		require.Equal(t, []string{"und", "m", "kg", "l", "caja", "rollo", "bulto", "galon"}, codes)
		require.NotContains(t, tpl.Modules, modules.Variants)
		require.Empty(t, tpl.VariantAxes)
	})

	t.Run("ropa y calzado: trae variantes con los ejes talla y color", func(t *testing.T) {
		tpl, ok := catalog.Get("ropa_calzado")
		require.True(t, ok)
		require.Contains(t, tpl.Modules, modules.Variants)
		axes := []string{}
		for _, a := range tpl.VariantAxes {
			axes = append(axes, a.Name)
			require.NotEmpty(t, a.Values, a.Name)
		}
		require.Subset(t, axes, []string{"talla", "color"})
	})

	t.Run("cada plantilla propone su IVA por defecto: droguería 0 y las demás 19 %", func(t *testing.T) {
		want := map[string]string{
			"generica": "0.19", "ferreteria": "0.19", "tienda": "0.19",
			"drogueria": "0", "papeleria": "0.19", "ropa_calzado": "0.19",
		}
		for _, tpl := range catalog.List() {
			require.True(t, decimal.RequireFromString(want[tpl.ID]).Equal(tpl.DefaultTaxRate), "%s: %s", tpl.ID, tpl.DefaultTaxRate)
		}
	})

	t.Run("una plantilla inexistente no se encuentra", func(t *testing.T) {
		_, ok := catalog.Get("panaderia")
		require.False(t, ok)
	})
}

// valid is a minimal valid template; each case breaks it.
const valid = `
name: Negocio de prueba
description: Una plantilla para los tests.
modules: [variants]
default_tax_rate: "0.19"
units:
  - {code: und, name: Unidad}
  - {code: par, name: Par}
categories: [General]
attributes:
  - {key: marca, label: Marca, type: text, required: true}
  - {key: genero, label: Género, type: list, options: [Mujer, Hombre]}
variant_axes:
  - {name: talla, values: [S, M]}
customer_segments: [Público, Mayorista]
`

func TestLoad(t *testing.T) {
	load := func(files map[string]string, checkModules func([]string) error) error {
		fsys := fstest.MapFS{}
		for name, content := range files {
			fsys[name] = &fstest.MapFile{Data: []byte(content)}
		}
		_, err := templates.Load(fsys, checkModules)
		return err
	}
	allModules := func([]string) error { return nil }

	t.Run("carga una plantilla válida con todos sus datos", func(t *testing.T) {
		catalog, err := templates.Load(fstest.MapFS{"prueba.yaml": {Data: []byte(valid)}}, allModules)
		require.NoError(t, err)
		tpl, ok := catalog.Get("prueba")
		require.True(t, ok)
		require.Equal(t, templates.Template{
			ID: "prueba", Name: "Negocio de prueba", Description: "Una plantilla para los tests.",
			Modules:        []string{"variants"},
			DefaultTaxRate: decimal.RequireFromString("0.19"),
			Units:          []templates.Unit{{Code: "und", Name: "Unidad"}, {Code: "par", Name: "Par"}},
			Categories:     []string{"General"},
			Attributes: []templates.Attribute{
				{Key: "marca", Label: "Marca", Type: "text", Required: true},
				{Key: "genero", Label: "Género", Type: "list", Options: []string{"Mujer", "Hombre"}},
			},
			VariantAxes:      []templates.Axis{{Name: "talla", Values: []string{"S", "M"}}},
			CustomerSegments: []string{"Público", "Mayorista"},
		}, tpl)
		require.Equal(t, []templates.Template{tpl}, catalog.List())
	})

	t.Run("un atributo sin required no es obligatorio", func(t *testing.T) {
		catalog, err := templates.Load(fstest.MapFS{"prueba.yaml": {Data: []byte(replace(valid, "type: text, required: true}", "type: text}"))}}, allModules)
		require.NoError(t, err)
		tpl, _ := catalog.Get("prueba")
		require.False(t, tpl.Attributes[0].Required)
	})

	t.Run("rechaza plantillas que no cumplen el esquema o sus reglas", func(t *testing.T) {
		tests := []struct {
			name    string
			files   map[string]string
			wantErr error
		}{
			{name: "sin plantillas", files: map[string]string{}, wantErr: templates.ErrNoTemplates},
			{name: "YAML mal formado", files: map[string]string{"prueba.yaml": "name: [sin cerrar"}},
			{name: "campo desconocido", files: map[string]string{"prueba.yaml": valid + "colores: [rojo]\n"}},
			{name: "sin unidades", files: map[string]string{"prueba.yaml": replace(valid, "  - {code: und, name: Unidad}\n  - {code: par, name: Par}\n", "")}},
			{name: "unidad repetida", files: map[string]string{"prueba.yaml": replace(valid, "{code: par, name: Par}", "{code: und, name: Otra}")}},
			{name: "categoría repetida", files: map[string]string{"prueba.yaml": replace(valid, "[General]", "[General, General]")}},
			{name: "atributo de lista sin opciones", files: map[string]string{"prueba.yaml": replace(valid, ", options: [Mujer, Hombre]", "")}},
			{name: "atributo de texto con opciones", files: map[string]string{"prueba.yaml": replace(valid, "type: text,", "type: text, options: [A],")}},
			{name: "atributo de tipo desconocido", files: map[string]string{"prueba.yaml": replace(valid, "type: text,", "type: date,")}},
			{name: "atributo con required que no es sí/no", files: map[string]string{"prueba.yaml": replace(valid, "required: true", "required: si")}},
			{name: "atributo con clave repetida", files: map[string]string{"prueba.yaml": replace(valid, "key: genero", "key: marca")}},
			{name: "sin IVA por defecto", files: map[string]string{"prueba.yaml": replace(valid, "default_tax_rate: \"0.19\"\n", "")}},
			{name: "IVA por defecto de 100 %", files: map[string]string{"prueba.yaml": replace(valid, `"0.19"`, `"1"`)}},
			{name: "IVA por defecto con más de 4 decimales", files: map[string]string{"prueba.yaml": replace(valid, `"0.19"`, `"0.12345"`)}},
			{name: "IVA por defecto como número y no como texto", files: map[string]string{"prueba.yaml": replace(valid, `"0.19"`, `0.19`)}},
			{name: "eje sin valores", files: map[string]string{"prueba.yaml": replace(valid, "values: [S, M]", "values: []")}},
			{name: "eje repetido", files: map[string]string{"prueba.yaml": replace(valid, "  - {name: talla, values: [S, M]}\n", "  - {name: talla, values: [S, M]}\n  - {name: Talla, values: [L]}\n")}},
			{name: "segmento repetido", files: map[string]string{"prueba.yaml": replace(valid, "[Público, Mayorista]", "[Público, Público]")}},
			{name: "nombre de archivo que no es un ID", files: map[string]string{"Ropa-Calzado.yaml": valid}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := load(tt.files, allModules)
				require.Error(t, err)
				if tt.wantErr != nil {
					require.ErrorIs(t, err, tt.wantErr)
				} else {
					require.ErrorIs(t, err, templates.ErrInvalidTemplate)
				}
			})
		}
	})

	t.Run("rechaza una plantilla con un módulo no registrado o sin su dependencia", func(t *testing.T) {
		boom := errors.New("modules: unknown module")
		var got []string
		err := load(map[string]string{"prueba.yaml": valid}, func(ids []string) error { got = ids; return boom })
		require.ErrorIs(t, err, boom)
		require.ErrorContains(t, err, "prueba")
		require.Equal(t, []string{"variants"}, got)
	})

	t.Run("con el registro de Cuadra, rechaza un módulo que no existe", func(t *testing.T) {
		err := load(map[string]string{"prueba.yaml": replace(valid, "[variants]", "[lots_expiry]")}, builtinModules(t))
		require.ErrorIs(t, err, modules.ErrUnknownModule)
	})

	t.Run("ignora los archivos que no son YAML", func(t *testing.T) {
		catalog, err := templates.Load(fstest.MapFS{
			"prueba.yaml": {Data: []byte(valid)},
			"embed.go":    {Data: []byte("package business")},
		}, allModules)
		require.NoError(t, err)
		require.Len(t, catalog.List(), 1)
	})
}

// replace replaces old, which must appear exactly once in s, with repl.
func replace(s, old, repl string) string {
	if strings.Count(s, old) != 1 {
		panic("replace: " + old + " does not appear exactly once")
	}
	return strings.Replace(s, old, repl, 1)
}
