package main

import (
	"bytes"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Rules say how long a Go file may be.
type Rules struct {
	MaxLines     int
	MaxTestLines int // for _test.go files
	// Allow lists files that may pass the limit for now. The list only
	// shrinks: an entry that no longer passes the limit must be removed.
	Allow []string
}

// Finding is a file longer than its limit.
type Finding struct {
	Path  string
	Lines int
	Max   int
}

// Result holds the findings, sorted by path, and the entries of Allow that
// are no longer needed.
type Result struct {
	Findings []Finding
	Unused   []string
}

// generatedHeader is the marker of generated Go code (go help generate).
var generatedHeader = regexp.MustCompile(`(?m)^// Code generated .* DO NOT EDIT\.$`)

// skippedDirs hold code that is not written by hand or not part of the build.
var skippedDirs = []string{"testdata", "vendor", "sqlcgen", "node_modules"}

// Scan walks fsys and reports every hand-written Go file over its limit.
func Scan(fsys fs.FS, rules Rules) (Result, error) {
	var result Result
	over := map[string]bool{}
	err := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return skipDir(entry.Name())
		}
		finding, long, err := check(fsys, name, rules)
		if err != nil || !long {
			return err
		}
		over[name] = true
		if !slices.Contains(rules.Allow, name) {
			result.Findings = append(result.Findings, finding)
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	for _, name := range rules.Allow {
		if !over[name] {
			result.Unused = append(result.Unused, name)
		}
	}
	return result, nil
}

func skipDir(name string) error {
	if slices.Contains(skippedDirs, name) {
		return fs.SkipDir
	}
	return nil
}

// check reports whether a hand-written Go file passes its limit, and by how
// much. Other files are not checked.
func check(fsys fs.FS, name string, rules Rules) (Finding, bool, error) {
	if path.Ext(name) != ".go" || strings.HasSuffix(name, ".gen.go") {
		return Finding{}, false, nil
	}
	data, err := fs.ReadFile(fsys, name)
	if err != nil || isGenerated(data) {
		return Finding{}, false, err
	}
	limit := rules.MaxLines
	if strings.HasSuffix(name, "_test.go") {
		limit = rules.MaxTestLines
	}
	if n := countLines(data); n > limit {
		return Finding{Path: name, Lines: n, Max: limit}, true, nil
	}
	return Finding{}, false, nil
}

// isGenerated looks for the marker before the package clause, where Go
// expects it.
func isGenerated(data []byte) bool {
	head, _, _ := bytes.Cut(append([]byte("\n"), data...), []byte("\npackage "))
	return generatedHeader.Match(head)
}

func countLines(data []byte) int {
	n := bytes.Count(data, []byte("\n"))
	if len(data) > 0 && !bytes.HasSuffix(data, []byte("\n")) {
		n++
	}
	return n
}
