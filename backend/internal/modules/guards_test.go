// Package modules_test checks the conventions of the modules that no linter
// covers (docs/backend.md): the depguard rules of every module, a span per
// use case and the names and attributes of the metrics.
package modules_test

import (
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

const modulesPkg = "github.com/brayangomez22/cuadra-app/backend/internal/modules/"

type depguardRule struct {
	Files []string    `yaml:"files"`
	Deny  []denyEntry `yaml:"deny"`
}

type denyEntry struct {
	Pkg string `yaml:"pkg"`
}

type lintConfig struct {
	Linters struct {
		Settings struct {
			Depguard struct {
				Rules map[string]depguardRule `yaml:"rules"`
			} `yaml:"depguard"`
		} `yaml:"settings"`
	} `yaml:"linters"`
}

func (r depguardRule) denies(pkg string) bool {
	return slices.ContainsFunc(r.Deny, func(d denyEntry) bool { return d.Pkg == pkg })
}

// The depguard rules of .golangci.yml name the modules one by one, so a new
// module without its rules could import another one unnoticed.

// missingGuards lists what the rules lack for modules: a rule per module,
// on its files, that denies every other module, and the app rule denying
// each module's adapters.
func missingGuards(rules map[string]depguardRule, modules []string) []string {
	var missing []string
	for _, m := range modules {
		if !rules["app"].denies(modulesPkg + m + "/adapters") {
			missing = append(missing, "la regla app no prohíbe "+m+"/adapters")
		}
		rule, ok := rules[m]
		if !ok || !slices.Contains(rule.Files, "**/internal/modules/"+m+"/**") {
			missing = append(missing, "regla "+m+" sobre **/internal/modules/"+m+"/**")
			continue
		}
		for _, other := range modules {
			if other != m && !rule.denies(modulesPkg+other) {
				missing = append(missing, "la regla "+m+" no prohíbe importar "+other)
			}
		}
	}
	return missing
}

func loadRules(t *testing.T) map[string]depguardRule {
	t.Helper()
	data, err := os.ReadFile("../../.golangci.yml")
	require.NoError(t, err)
	var cfg lintConfig
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	return cfg.Linters.Settings.Depguard.Rules
}

func moduleDirs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	var modules []string
	for _, e := range entries {
		if e.IsDir() {
			modules = append(modules, e.Name())
		}
	}
	return modules
}

func TestDepguardCoversEveryModule(t *testing.T) {
	t.Run("cada módulo de internal/modules tiene sus reglas de depguard", func(t *testing.T) {
		require.Empty(t, missingGuards(loadRules(t), moduleDirs(t)),
			"agrega las reglas del módulo a .golangci.yml (docs/backend.md, \"Agregar un módulo\")")
	})

	t.Run("un módulo nuevo sin reglas se detecta", func(t *testing.T) {
		modules := append(moduleDirs(t), "example")

		missing := missingGuards(loadRules(t), modules)

		require.Contains(t, missing, "regla example sobre **/internal/modules/example/**")
		require.Contains(t, missing, "la regla sales no prohíbe importar example")
		require.Contains(t, missing, "la regla app no prohíbe example/adapters")
	})
}
