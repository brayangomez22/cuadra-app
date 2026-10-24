// Command lint-size fails if a hand-written Go file of the backend is longer
// than its limit (make lint-size): 300 lines of code, 400 of tests. Long
// files are split by use case, resource or scenario (docs/backend.md).
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	root := flag.String("root", ".", "backend directory")
	flag.Parse()

	result, err := Scan(os.DirFS(*root), DefaultRules())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, f := range result.Findings {
		fmt.Printf("%s: %d líneas (máximo %d): divídalo por caso de uso, recurso o escenario\n", f.Path, f.Lines, f.Max)
	}
	for _, name := range result.Unused {
		fmt.Printf("%s: ya cumple el límite: quítelo de la lista temporal de DefaultRules\n", name)
	}
	if len(result.Findings) > 0 || len(result.Unused) > 0 {
		os.Exit(1)
	}
}

// DefaultRules are the limits make lint-size applies to the backend.
func DefaultRules() Rules {
	return Rules{
		MaxLines:     300,
		MaxTestLines: 400,
	}
}
