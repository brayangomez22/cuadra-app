// Command lint-neutral fails if a term of one business type appears in the
// core (make lint-neutral). The core is generic: what belongs to one kind of
// business lives in templates, modules or tenant settings (CLAUDE.md).
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()

	result, err := Scan(os.DirFS(*root), DefaultRules())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, f := range result.Findings {
		fmt.Printf("%s:%d: término de un nicho %q en el core\n", f.Path, f.Line, f.Term)
	}
	for _, a := range result.Unused {
		fmt.Printf("%s: la excepción de %q ya no se usa (%s): quítela de DefaultRules\n", a.Path, a.Term, a.Reason)
	}
	if len(result.Findings) > 0 || len(result.Unused) > 0 {
		os.Exit(1)
	}
}
