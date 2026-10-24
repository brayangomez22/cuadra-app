package modules_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// noSpanMarker is how a use case says, in its comment, why it opens no span
// of its own (it runs on every request, or it reads static data).
const noSpanMarker = "opens no span"

// useCasesWithoutSpan lists the exported methods of *Service in the app
// packages under root that neither open a span nor say why in their doc.
func useCasesWithoutSpan(t *testing.T, root string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(root, "*", "app", "*.go"))
	require.NoError(t, err)
	var missing []string
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ParseComments)
		require.NoError(t, err)
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !isServiceUseCase(fn) || opensSpan(fn) || strings.Contains(fn.Doc.Text(), noSpanMarker) {
				continue
			}
			missing = append(missing, name+": "+fn.Name.Name)
		}
	}
	return missing
}

// isServiceUseCase reports whether fn is an exported method of *Service.
func isServiceUseCase(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) != 1 || !fn.Name.IsExported() {
		return false
	}
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	id, ok := star.X.(*ast.Ident)
	return ok && id.Name == "Service"
}

// opensSpan reports whether fn calls s.startSpan or s.spans.Start(For).
func opensSpan(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return !found
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			switch sel.Sel.Name {
			case "startSpan", "Start", "StartFor":
				found = true
			}
		}
		return !found
	})
	return found
}

func TestEveryUseCaseOpensASpan(t *testing.T) {
	t.Run("cada caso de uso de los módulos abre su span <modulo>.<CasoDeUso>", func(t *testing.T) {
		require.Empty(t, useCasesWithoutSpan(t, "."),
			"abre el span con s.startSpan (o di en su comentario por qué %s)", noSpanMarker)
	})
}
