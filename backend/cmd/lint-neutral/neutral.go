package main

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Rules say which terms are forbidden and where.
type Rules struct {
	// Terms are matched case-insensitively at the start of a word, so a term
	// covers its endings ("ferreter" matches "Ferretería"). Words in a
	// multi-word term may be joined by spaces, "_", "-" or nothing.
	Terms []string
	// Roots are the core: files or directories, relative to the repository.
	Roots []string
	// Extensions are the file types scanned under Roots.
	Extensions []string
	// ExcludeSuffixes, ExcludeDirs (a directory name at any depth) and
	// ExcludePaths (a path prefix) leave files out of the scan.
	ExcludeSuffixes []string
	ExcludeDirs     []string
	ExcludePaths    []string
	// Allow lists temporary exceptions, each tied to the pending item that
	// removes it.
	Allow []Allowance
}

// Allowance lets one term appear in one file until Reason is done.
type Allowance struct {
	Path   string
	Term   string
	Reason string
}

// Finding is a forbidden term in a core file.
type Finding struct {
	Path string
	Line int
	Term string
}

// Result holds the findings, sorted by path and line, and the allowances
// that matched nothing (they must be removed).
type Result struct {
	Findings []Finding
	Unused   []Allowance
}

// DefaultRules are the rules make lint-neutral applies to the repository.
// Business templates, migrations and the demo seed are not under Roots.
func DefaultRules() Rules {
	return Rules{
		Terms: []string{
			"ferreter", "bulto", "rollo", "maestro de obra", "contratista", "talla",
			"tornillo", "cemento", "varilla", "plomer", "droguer", "papeler", "calzado",
		},
		Roots: []string{
			"api/openapi.yaml",
			"backend/cmd",
			"backend/internal/modules",
			"backend/internal/platform",
			"backend/internal/server",
			"frontend/src",
		},
		Extensions:      []string{".go", ".sql", ".yaml", ".yml", ".json", ".ts", ".tsx", ".scss", ".css", ".html"},
		ExcludeSuffixes: []string{"_test.go", ".test.ts", ".test.tsx"},
		ExcludeDirs:     []string{"testdata", "fixtures", "testing"}, // testing/: the fixtures of a frontend feature (docs/frontend.md)
		ExcludePaths: []string{
			"frontend/src/test/",         // test fixtures and the fake API
			"frontend/src/features/dev/", // UI catalog: sample data, only in development
			"backend/cmd/lint-neutral/",  // the term list itself
		},
	}
}

type matcher struct {
	term string
	re   *regexp.Regexp
}

// Scan walks the roots of fsys and reports every forbidden term.
func Scan(fsys fs.FS, rules Rules) (Result, error) {
	matchers, err := compileTerms(rules.Terms)
	if err != nil {
		return Result{}, err
	}
	s := &scanner{fsys: fsys, rules: rules, matchers: matchers, used: make([]bool, len(rules.Allow))}
	for _, root := range rules.Roots {
		if err := fs.WalkDir(fsys, root, s.visit(root)); err != nil {
			return Result{}, err
		}
	}
	return s.result(), nil
}

// compileTerms builds the matcher of each term: case-insensitive, with the
// words of a multi-word term joined by spaces, "_", "-" or nothing.
func compileTerms(terms []string) ([]matcher, error) {
	matchers := make([]matcher, 0, len(terms))
	for _, term := range terms {
		words := strings.Fields(term)
		for i, word := range words {
			words[i] = regexp.QuoteMeta(word)
		}
		re, err := regexp.Compile("(?i)" + strings.Join(words, `[\s_-]*`))
		if err != nil {
			return nil, fmt.Errorf("term %q: %w", term, err)
		}
		matchers = append(matchers, matcher{term: term, re: re})
	}
	return matchers, nil
}

// scanner collects the findings of a Scan and which allowances matched.
type scanner struct {
	fsys     fs.FS
	rules    Rules
	matchers []matcher
	used     []bool
	findings []Finding
}

// visit walks one root: a missing root is skipped.
func (s *scanner) visit(root string) fs.WalkDirFunc {
	return func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			if name == root && errors.Is(err, fs.ErrNotExist) {
				return fs.SkipDir
			}
			return err
		}
		if entry.IsDir() {
			return s.enter(name, entry.Name())
		}
		if !scanned(s.rules, name) {
			return nil
		}
		return s.scan(name)
	}
}

// enter skips the excluded directories.
func (s *scanner) enter(name, dirName string) error {
	if slices.Contains(s.rules.ExcludeDirs, dirName) || excludedPath(s.rules, name+"/") {
		return fs.SkipDir
	}
	return nil
}

// scan keeps the findings of a file that no allowance covers.
func (s *scanner) scan(name string) error {
	data, err := fs.ReadFile(s.fsys, name)
	if err != nil {
		return err
	}
	for _, finding := range scanFile(name, data, s.matchers) {
		if i := allowed(s.rules.Allow, finding); i >= 0 {
			s.used[i] = true
			continue
		}
		s.findings = append(s.findings, finding)
	}
	return nil
}

// result sorts the findings by path and line and lists the allowances that
// matched nothing.
func (s *scanner) result() Result {
	slices.SortFunc(s.findings, func(a, b Finding) int {
		if c := strings.Compare(a.Path, b.Path); c != 0 {
			return c
		}
		return a.Line - b.Line
	})
	var unused []Allowance
	for i, allowance := range s.rules.Allow {
		if !s.used[i] {
			unused = append(unused, allowance)
		}
	}
	return Result{Findings: s.findings, Unused: unused}
}

func scanned(rules Rules, name string) bool {
	if !slices.Contains(rules.Extensions, path.Ext(name)) || excludedPath(rules, name) {
		return false
	}
	for _, suffix := range rules.ExcludeSuffixes {
		if strings.HasSuffix(name, suffix) {
			return false
		}
	}
	return true
}

func excludedPath(rules Rules, name string) bool {
	for _, prefix := range rules.ExcludePaths {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func allowed(allow []Allowance, finding Finding) int {
	for i, allowance := range allow {
		if allowance.Path == finding.Path && allowance.Term == finding.Term {
			return i
		}
	}
	return -1
}

// scanFile reports each term once per line.
func scanFile(name string, data []byte, matchers []matcher) []Finding {
	var findings []Finding
	for i, line := range strings.Split(string(data), "\n") {
		number := i + 1
		for _, m := range matchers {
			for _, loc := range m.re.FindAllStringIndex(line, -1) {
				if startsWord(line, loc[0]) {
					findings = append(findings, Finding{Path: name, Line: number, Term: m.term})
					break
				}
			}
		}
	}
	return findings
}

// startsWord reports whether the match at i begins a word: it follows the
// start of the line, a non-letter ("ropa_calzado") or a lowercase letter
// when the match itself is uppercase ("isFerreteria"). A match inside a
// word ("pantalla", "totalLabel") does not count.
func startsWord(line string, i int) bool {
	if i == 0 {
		return true
	}
	prev, _ := utf8.DecodeLastRuneInString(line[:i])
	if !unicode.IsLetter(prev) {
		return true
	}
	first, _ := utf8.DecodeRuneInString(line[i:])
	return unicode.IsLower(prev) && unicode.IsUpper(first)
}
