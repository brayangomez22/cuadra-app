package modules_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// metricName is cuadra.<module>.<thing> in snake_case.
var metricName = regexp.MustCompile(`^cuadra\.([a-z]+)\.[a-z][a-z0-9_]*$`)

// badMetrics lists the instruments created with usecase.Counter or
// usecase.Histogram in the app packages under root whose name is not
// cuadra.<its module>.<thing> or that have no unit, and the measurements
// that carry the tenant or the user as an attribute.
func badMetrics(t *testing.T, root string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(root, "*", "app", "*.go"))
	require.NoError(t, err)
	var bad []string
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		module := strings.Split(filepath.ToSlash(name), "/")[0]
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		require.NoError(t, err)
		ast.Inspect(f, func(n ast.Node) bool {
			for _, problem := range []string{checkInstrument(n, module), checkAttributes(n)} {
				if problem != "" {
					bad = append(bad, name+": "+problem)
				}
			}
			return true
		})
	}
	return bad
}

// checkInstrument returns what is wrong with n if it creates an instrument:
// usecase.Counter(&target, name, description, unit, ...).
func checkInstrument(n ast.Node, module string) string {
	call, ok := n.(*ast.CallExpr)
	if !ok || len(call.Args) < 4 {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || (sel.Sel.Name != "Counter" && sel.Sel.Name != "Histogram") {
		return ""
	}
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "usecase" {
		return ""
	}
	metric, unit := literal(call.Args[1]), literal(call.Args[3])
	if m := metricName.FindStringSubmatch(metric); m == nil || m[1] != module {
		return metric + " no es cuadra." + module + ".<cosa>"
	}
	if unit == "" {
		return metric + " no tiene unidad"
	}
	return ""
}

// highCardinality are the attribute keys a metric must never carry: one
// series per tenant or user (CLAUDE.md, "Observabilidad"). Traces and logs
// carry them.
var highCardinality = regexp.MustCompile(`^(tenant|user)\.`)

// checkAttributes returns what is wrong with n if it is a
// metric.WithAttributes(...) with a tenant or user attribute.
func checkAttributes(n ast.Node) string {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "WithAttributes" {
		return ""
	}
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "metric" {
		return ""
	}
	problem := ""
	ast.Inspect(call, func(n ast.Node) bool {
		if key := literal(asExpr(n)); highCardinality.MatchString(key) {
			problem = "la métrica lleva el atributo " + key
		}
		return problem == ""
	})
	return problem
}

func asExpr(n ast.Node) ast.Expr {
	e, _ := n.(ast.Expr)
	return e
}

func literal(e ast.Expr) string {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return ""
	}
	return s
}

func TestMetricNames(t *testing.T) {
	t.Run("cada métrica de un módulo se llama cuadra.<modulo>.<cosa>, tiene unidad y no lleva tenant ni usuario", func(t *testing.T) {
		require.Empty(t, badMetrics(t, "."))
	})
}
