// Package templates reads the business templates: the data a new tenant
// starts with, by kind of business (backend/templates/business/*.yaml).
// Templates are data, not code: a template never adds behavior, it only
// declares the modules, units, categories, product attributes, variant axes
// and customer segments a tenant gets at sign-up. The tenant can change all
// of it afterwards.
//
// Load validates every template against schema.json and against the rules
// a schema cannot express (no repeated units, keys or names; registered
// modules with their dependencies). It runs at startup, so a broken
// template stops the API instead of failing every sign-up.
package templates

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

// Errors of Load.
var (
	ErrNoTemplates     = errors.New("templates: no business templates")
	ErrInvalidTemplate = errors.New("templates: invalid business template")
)

//go:embed schema.json
var schemaJSON []byte

// schemaURL is the schema's $id: a name, never fetched.
const schemaURL = "https://cuadra.app/schemas/business-template.json"

// idFormat is the format of a template ID, which is its file name: the same
// as tenants.business_template accepts.
var idFormat = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// Template is a business template.
type Template struct {
	// ID is the file name without ".yaml" ("generica").
	ID string
	// Name and Description are shown to whoever signs up.
	Name        string
	Description string
	// Modules are the modules active from the sign-up.
	Modules    []string
	Units      []Unit
	Categories []string
	Attributes []Attribute
	// VariantAxes are the axes the tenant's variants vary on (size,
	// color...).
	VariantAxes []Axis
	// CustomerSegments are declared and validated; they are seeded once
	// customers exist (T31).
	CustomerSegments []string
}

// Unit is a unit of measure of a template.
type Unit struct {
	Code string `yaml:"code"`
	Name string `yaml:"name"`
}

// Attribute is a product attribute definition of a template. Type is one of
// the catalog's attribute types (text, number, list, boolean); Options are
// for list attributes only.
type Attribute struct {
	Key     string   `yaml:"key"`
	Label   string   `yaml:"label"`
	Type    string   `yaml:"type"`
	Options []string `yaml:"options"`
}

// Axis is a variant axis of a template and the values it allows.
type Axis struct {
	Name   string   `yaml:"name"`
	Values []string `yaml:"values"`
}

// file is a template as its YAML spells it.
type file struct {
	Name             string      `yaml:"name"`
	Description      string      `yaml:"description"`
	Modules          []string    `yaml:"modules"`
	Units            []Unit      `yaml:"units"`
	Categories       []string    `yaml:"categories"`
	Attributes       []Attribute `yaml:"attributes"`
	VariantAxes      []Axis      `yaml:"variant_axes"`
	CustomerSegments []string    `yaml:"customer_segments"`
}

// Catalog is the set of loaded templates. It never changes after Load.
type Catalog struct {
	list []Template
}

// Load reads every *.yaml at the root of fsys as a template and validates
// it; checkModules validates each template's modules (the module registry's
// CheckSet). It fails if there is no template or any of them is invalid.
func Load(fsys fs.FS, checkModules func([]string) error) (*Catalog, error) {
	schema, err := compileSchema()
	if err != nil {
		return nil, err
	}
	names, err := fs.Glob(fsys, "*.yaml")
	if err != nil {
		return nil, fmt.Errorf("templates: list: %w", err)
	}
	if len(names) == 0 {
		return nil, ErrNoTemplates
	}
	c := &Catalog{}
	for _, name := range names {
		t, err := load(fsys, name, schema)
		if err != nil {
			return nil, err
		}
		if err := checkModules(t.Modules); err != nil {
			return nil, fmt.Errorf("templates: %s: %w", t.ID, err)
		}
		c.list = append(c.list, t)
	}
	return c, nil
}

func compileSchema() (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("templates: read schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaURL, doc); err != nil {
		return nil, fmt.Errorf("templates: add schema: %w", err)
	}
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("templates: compile schema: %w", err)
	}
	return schema, nil
}

// load reads and validates the template in name.
func load(fsys fs.FS, name string, schema *jsonschema.Schema) (Template, error) {
	id := strings.TrimSuffix(path.Base(name), ".yaml")
	invalid := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s: %s", ErrInvalidTemplate, name, fmt.Sprintf(format, args...))
	}
	if !idFormat.MatchString(id) {
		return Template{}, invalid("the file name must be an ID (lowercase letters, digits and underscores)")
	}
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return Template{}, fmt.Errorf("templates: read %s: %w", name, err)
	}

	var doc any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return Template{}, invalid("%v", err)
	}
	if err := schema.Validate(doc); err != nil {
		// The detailed form names each failing field ("at '/units/1'").
		return Template{}, invalid("%#v", err)
	}
	var f file
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil {
		return Template{}, invalid("%v", err)
	}

	t := Template{
		ID: id, Name: f.Name, Description: f.Description, Modules: f.Modules, Units: f.Units,
		Categories: f.Categories, Attributes: f.Attributes, VariantAxes: f.VariantAxes, CustomerSegments: f.CustomerSegments,
	}
	if err := checkUnique(t); err != nil {
		return Template{}, invalid("%v", err)
	}
	return t, nil
}

// checkUnique rejects repeated units, categories, attribute keys, variant
// axes and segments. Names compare trimmed and case-insensitive, as the
// catalog stores and matches them.
func checkUnique(t Template) error {
	checks := []struct {
		what string
		keys []string
	}{
		{"unit", keys(t.Units, func(u Unit) string { return u.Code })},
		{"category", t.Categories},
		{"attribute", keys(t.Attributes, func(a Attribute) string { return a.Key })},
		{"variant axis", keys(t.VariantAxes, func(a Axis) string { return a.Name })},
		{"customer segment", t.CustomerSegments},
	}
	for _, c := range checks {
		seen := make([]string, 0, len(c.keys))
		for _, k := range c.keys {
			k = strings.ToLower(strings.TrimSpace(k))
			if slices.Contains(seen, k) {
				return fmt.Errorf("repeated %s %q", c.what, k)
			}
			seen = append(seen, k)
		}
	}
	return nil
}

func keys[T any](items []T, key func(T) string) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, key(it))
	}
	return out
}

// List returns every template, by ID.
func (c *Catalog) List() []Template {
	out := make([]Template, len(c.list))
	for i, t := range c.list {
		out[i] = t.clone()
	}
	return out
}

// Get returns the template id; ok is false when there is none.
func (c *Catalog) Get(id string) (Template, bool) {
	for _, t := range c.list {
		if t.ID == id {
			return t.clone(), true
		}
	}
	return Template{}, false
}

// clone copies t so a caller cannot change the catalog.
func (t Template) clone() Template {
	t.Modules = slices.Clone(t.Modules)
	t.Units = slices.Clone(t.Units)
	t.Categories = slices.Clone(t.Categories)
	t.CustomerSegments = slices.Clone(t.CustomerSegments)
	attrs := make([]Attribute, len(t.Attributes))
	for i, a := range t.Attributes {
		a.Options = slices.Clone(a.Options)
		attrs[i] = a
	}
	t.Attributes = attrs
	axes := make([]Axis, len(t.VariantAxes))
	for i, a := range t.VariantAxes {
		a.Values = slices.Clone(a.Values)
		axes[i] = a
	}
	t.VariantAxes = axes
	return t
}
