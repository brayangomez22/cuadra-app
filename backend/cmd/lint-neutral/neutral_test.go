package main

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func file(content string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(content)} }

// testRules scans everything under core/ with two terms, so each test sets
// only the files it needs.
func testRules() Rules {
	return Rules{
		Terms:      []string{"ferreter", "talla", "maestro de obra"},
		Roots:      []string{"core"},
		Extensions: []string{".go", ".ts", ".tsx", ".yaml"},
	}
}

func scan(t *testing.T, fsys fstest.MapFS, rules Rules) Result {
	t.Helper()
	result, err := Scan(fsys, rules)
	require.NoError(t, err)
	return result
}

func TestScan(t *testing.T) {
	t.Run("detecta un término prohibido en un archivo del core con su línea", func(t *testing.T) {
		fsys := fstest.MapFS{
			"core/catalog/product.go": file("package catalog\n\n// El catálogo de la ferretería.\n"),
		}

		result := scan(t, fsys, testRules())

		require.Equal(t, []Finding{{Path: "core/catalog/product.go", Line: 3, Term: "ferreter"}}, result.Findings)
	})

	t.Run("no distingue mayúsculas ni el resto de la palabra (Ferretería, TALLAS)", func(t *testing.T) {
		fsys := fstest.MapFS{
			"core/a.ts": file("const a = 'Ferretería El Centro';\nconst b = 'TALLAS';\n"),
		}

		result := scan(t, fsys, testRules())

		require.Equal(t, []Finding{
			{Path: "core/a.ts", Line: 1, Term: "ferreter"},
			{Path: "core/a.ts", Line: 2, Term: "talla"},
		}, result.Findings)
	})

	t.Run("detecta un término en camelCase o snake_case (isFerreteria, ropa_talla)", func(t *testing.T) {
		fsys := fstest.MapFS{
			"core/a.go": file("if isFerreteria {\n}\nconst ropa_talla = 1\n"),
		}

		result := scan(t, fsys, testRules())

		require.Equal(t, []Finding{
			{Path: "core/a.go", Line: 1, Term: "ferreter"},
			{Path: "core/a.go", Line: 3, Term: "talla"},
		}, result.Findings)
	})

	t.Run("detecta un término de varias palabras con espacios, guiones o sin separador", func(t *testing.T) {
		fsys := fstest.MapFS{
			"core/a.ts": file("'Maestro de obra'\nmaestro_de_obra\nmaestroDeObra\nmaestro-de-obra\n"),
		}

		result := scan(t, fsys, testRules())

		require.Len(t, result.Findings, 4)
		for i, finding := range result.Findings {
			require.Equal(t, Finding{Path: "core/a.ts", Line: i + 1, Term: "maestro de obra"}, finding)
		}
	})

	t.Run("no marca un término dentro de otra palabra (pantalla, totalLabel)", func(t *testing.T) {
		fsys := fstest.MapFS{
			"core/a.tsx": file("<p>Recargue esta pantalla</p>\n<span className={styles.totalLabel} />\n"),
		}

		result := scan(t, fsys, testRules())

		require.Empty(t, result.Findings)
	})

	t.Run("no revisa tipos de archivo que no están en las reglas", func(t *testing.T) {
		fsys := fstest.MapFS{
			"core/logo.png": file("ferretería"),
		}

		result := scan(t, fsys, testRules())

		require.Empty(t, result.Findings)
	})

	t.Run("una excepción permite solo su término en su archivo", func(t *testing.T) {
		rules := testRules()
		rules.Allow = []Allowance{{Path: "core/units.ts", Term: "talla", Reason: "P33"}}
		fsys := fstest.MapFS{
			"core/units.ts": file("talla\nferretería\n"),
			"core/other.ts": file("talla\n"),
		}

		result := scan(t, fsys, rules)

		require.Equal(t, []Finding{
			{Path: "core/other.ts", Line: 1, Term: "talla"},
			{Path: "core/units.ts", Line: 2, Term: "ferreter"},
		}, result.Findings)
		require.Empty(t, result.Unused)
	})

	t.Run("reporta una excepción que ya no se usa", func(t *testing.T) {
		rules := testRules()
		allowance := Allowance{Path: "core/units.ts", Term: "talla", Reason: "P33"}
		rules.Allow = []Allowance{allowance}
		fsys := fstest.MapFS{
			"core/units.ts": file("und\n"),
		}

		result := scan(t, fsys, rules)

		require.Empty(t, result.Findings)
		require.Equal(t, []Allowance{allowance}, result.Unused)
	})
}

// The default rules are what make lint-neutral runs on the repository.
func TestDefaultRules(t *testing.T) {
	// Every allowance in DefaultRules must be used, so the fake repository
	// carries the files they cover.
	withAllowed := func(fsys fstest.MapFS) fstest.MapFS {
		content := map[string]string{}
		for _, allowance := range DefaultRules().Allow {
			content[allowance.Path] += allowance.Term + "\n"
		}
		for name, terms := range content {
			fsys[name] = file(terms)
		}
		return fsys
	}

	t.Run("detecta términos en cada ruta del core", func(t *testing.T) {
		fsys := withAllowed(fstest.MapFS{
			"api/openapi.yaml": file("example: Cemento gris 50kg\n"),
			"backend/internal/modules/catalog/domain/unit.go":         file("// bulto\n"),
			"backend/internal/modules/sales/adapters/http/handler.go": file("// talla\n"),
			"backend/internal/platform/templates/templates.go":        file("// ropa_calzado\n"),
			"backend/cmd/api/main.go":                                 file("// droguería\n"),
			"frontend/src/shared/ui/Input/Input.tsx":                  file("// Tornillo\n"),
			"frontend/src/features/catalog/ProductListPage.tsx":       file("'El catálogo de la ferretería'\n"),
		})

		result := scan(t, fsys, DefaultRules())

		require.Equal(t, []Finding{
			{Path: "api/openapi.yaml", Line: 1, Term: "cemento"},
			{Path: "backend/cmd/api/main.go", Line: 1, Term: "droguer"},
			{Path: "backend/internal/modules/catalog/domain/unit.go", Line: 1, Term: "bulto"},
			{Path: "backend/internal/modules/sales/adapters/http/handler.go", Line: 1, Term: "talla"},
			{Path: "backend/internal/platform/templates/templates.go", Line: 1, Term: "calzado"},
			{Path: "frontend/src/features/catalog/ProductListPage.tsx", Line: 1, Term: "ferreter"},
			{Path: "frontend/src/shared/ui/Input/Input.tsx", Line: 1, Term: "tornillo"},
		}, result.Findings)
		require.Empty(t, result.Unused)
	})

	t.Run("no marca los tests de Go ni de frontend (*_test.go, *.test.ts, *.test.tsx)", func(t *testing.T) {
		fsys := withAllowed(fstest.MapFS{
			"backend/internal/modules/catalog/domain/product_test.go": file("// Ferretería El Tornillo\n"),
			"frontend/src/features/sales/PosPage.test.tsx":            file("'Cemento gris 50kg'\n"),
			"frontend/src/features/sales/cart.test.ts":                file("'bulto'\n"),
		})

		result := scan(t, fsys, DefaultRules())

		require.Empty(t, result.Findings)
	})

	t.Run("no marca testdata, fixtures ni el catálogo de UI de desarrollo", func(t *testing.T) {
		fsys := withAllowed(fstest.MapFS{
			"backend/internal/platform/templates/testdata/ferreteria.yaml": file("name: Ferretería\n"),
			"frontend/src/test/fixtures.ts":                                file("'Tubo PVC'; 'bulto'\n"),
			"frontend/src/test/fakeApi.ts":                                 file("'Ferretería El Tornillo'\n"),
			"frontend/src/features/dev/UiCatalogPage.tsx":                  file("'Cemento gris 50kg'\n"),
		})

		result := scan(t, fsys, DefaultRules())

		require.Empty(t, result.Findings)
	})

	t.Run("no revisa rutas fuera del core (plantillas, migraciones)", func(t *testing.T) {
		fsys := withAllowed(fstest.MapFS{
			"backend/templates/business/ferreteria.yaml": file("units: [bulto, rollo]\n"),
			"backend/migrations/00004_catalog.sql":       file("CHECK (base_unit IN ('bulto', 'rollo'))\n"),
			"docs/negocio.md":                            file("Ferreterías\n"),
		})

		result := scan(t, fsys, DefaultRules())

		require.Empty(t, result.Findings)
	})

	t.Run("no se marca a sí mismo por la lista de términos", func(t *testing.T) {
		fsys := withAllowed(fstest.MapFS{
			"backend/cmd/lint-neutral/neutral.go": file(`var terms = []string{"ferreter", "bulto"}` + "\n"),
		})

		result := scan(t, fsys, DefaultRules())

		require.Empty(t, result.Findings)
	})
}
