// Package demo generates the demonstration tenants: realistic, repeatable
// data for one business template (a catalog, customers with credit,
// suppliers and six months of purchases, sales and payments) written
// through the API, and their nightly reset. It is not a business module:
// it is one more client of the contract, plus what internal/server's Demo
// offers that the contract does not (marking the tenant demo, its staff,
// purging it).
//
// The data of each business lives in data/<template>.yaml, outside the
// neutral core (lint-neutral does not scan this package).
package demo

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

//go:embed data/*.yaml
var dataFS embed.FS

// Profile is how a demo tenant of one business template looks: who it is,
// what it sells and how much, in a normal day.
type Profile struct {
	// Template is the business template the tenant signs up with (the file
	// name).
	Template  string
	Tenant    TenantProfile `yaml:"tenant"`
	Locations []string      `yaml:"locations"`
	// Products is how many articles (variants) the full catalog has.
	Products int `yaml:"products"`
	// Customers and the percentage of them with credit.
	Customers    int `yaml:"customers"`
	CreditShare  int `yaml:"credit_share"`
	DefaulterOne int `yaml:"defaulter_one_in"`
	// SalesPerDay is a normal day's sales per location; Weekdays (Sunday
	// first) and Months (January first) are percentages of it.
	SalesPerDay int     `yaml:"sales_per_day"`
	Weekdays    [7]int  `yaml:"weekdays"`
	Months      [12]int `yaml:"months"`
	// Segments are the customer segments of the template; credit customers
	// belong to CreditSegment.
	Segments      []string   `yaml:"segments"`
	CreditSegment string     `yaml:"credit_segment"`
	Suppliers     []Supplier `yaml:"suppliers"`
	Families      []Family   `yaml:"families"`
}

// TenantProfile is the demo business; Mailbox is the domain of its users'
// emails.
type TenantProfile struct {
	Name    string `yaml:"name"`
	NIT     string `yaml:"nit"`
	Mailbox string `yaml:"mailbox"`
}

// Supplier is a supplier of the demo tenant; NIT is the number without DV.
// Its invoices are numbered InvoicePrefix-000001 onwards.
type Supplier struct {
	Name          string `yaml:"name"`
	NIT           string `yaml:"nit"`
	TermDays      int    `yaml:"term_days"`
	InvoicePrefix string `yaml:"invoice_prefix"`
	ContactName   string `yaml:"contact_name"`
	Phone         string `yaml:"phone"`
	Email         string `yaml:"email"`
}

// Family is a kind of product: its articles combine a name, a spec and a
// brand, in the category and unit given, priced within Price (pesos).
type Family struct {
	Category   string            `yaml:"category"`
	Unit       string            `yaml:"unit"`
	Names      []string          `yaml:"names"`
	Specs      []string          `yaml:"specs"`
	Brands     []string          `yaml:"brands"`
	Attributes map[string]string `yaml:"attributes"`
	Price      [2]int            `yaml:"price"`
	// Quantity is the most a sale line takes; Step, when set, is the
	// fraction it is sold in ("0.5" meters).
	Quantity int    `yaml:"quantity"`
	Step     string `yaml:"step"`
	// Axes are the variant axes (name → values) the articles vary on; the
	// first is a range (sizes), the others a choice (colors).
	Axes []FamilyAxis `yaml:"axes"`
	// PurchaseUnit, when set, is the unit the suppliers sell it in (a box
	// of 12): its articles are bought in whole ones.
	PurchaseUnit *PurchaseUnit `yaml:"purchase_unit"`
}

// PurchaseUnit is an alternate unit of a family's articles and how many of
// its unit it brings.
type PurchaseUnit struct {
	Unit   string `yaml:"unit"`
	Factor int    `yaml:"factor"`
}

// FamilyAxis is a variant axis of a family and the values it uses.
type FamilyAxis struct {
	Name   string   `yaml:"name"`
	Values []string `yaml:"values"`
}

// Templates returns the business templates with a demo profile, sorted.
func Templates() []string {
	names, _ := fs.Glob(dataFS, "data/*.yaml")
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, strings.TrimSuffix(path.Base(n), ".yaml"))
	}
	slices.Sort(out)
	return out
}

// LoadProfile reads and checks the demo profile of a business template.
func LoadProfile(template string) (Profile, error) {
	if !slices.Contains(Templates(), template) {
		return Profile{}, fmt.Errorf("demo: no profile for template %q", template)
	}
	raw, err := dataFS.ReadFile("data/" + template + ".yaml")
	if err != nil {
		return Profile{}, err
	}
	p := Profile{Template: template}
	if err := yaml.Unmarshal(raw, &p); err != nil {
		return Profile{}, fmt.Errorf("demo: profile %s: %w", template, err)
	}
	if err := p.check(); err != nil {
		return Profile{}, fmt.Errorf("demo: profile %s: %w", template, err)
	}
	return p, nil
}

// check verifies what the generator relies on.
func (p Profile) check() error {
	if p.Tenant.Name == "" || p.Tenant.Mailbox == "" {
		return errors.New("tenant name and mailbox are required")
	}
	if slices.Contains([]int{len(p.Locations), len(p.Suppliers), len(p.Families), len(p.Segments)}, 0) {
		return errors.New("locations, suppliers, families and segments are required")
	}
	if min(p.Products, p.Customers, p.SalesPerDay) <= 0 {
		return errors.New("products, customers and sales_per_day must be positive")
	}
	if !slices.Contains(p.Segments, p.CreditSegment) {
		return errors.New("credit_segment must be one of the segments")
	}
	for _, s := range p.Suppliers {
		if err := s.check(); err != nil {
			return err
		}
	}
	for _, f := range p.Families {
		if err := errors.Join(f.check(), f.PurchaseUnit.check(f)); err != nil {
			return err
		}
	}
	return nil
}

// check verifies a supplier can number its invoices and give a due date.
func (s Supplier) check() error {
	if s.InvoicePrefix == "" || s.TermDays <= 0 {
		return fmt.Errorf("supplier %s: invoice_prefix and term_days are required", s.Name)
	}
	return nil
}

// check verifies a family can make articles.
func (f Family) check() error {
	if len(f.Names) == 0 || len(f.Specs) == 0 || len(f.Brands) == 0 {
		return fmt.Errorf("family %s: names, specs and brands are required", f.Category)
	}
	if f.Price[0] <= 0 || f.Price[1] < f.Price[0] || f.Quantity <= 0 {
		return fmt.Errorf("family %s: a price range and a quantity are required", f.Category)
	}
	return nil
}

// check verifies the purchase unit of f, if it has one, is another unit
// that brings more than one.
func (u *PurchaseUnit) check(f Family) error {
	if u != nil && (u.Unit == "" || u.Unit == f.Unit || u.Factor <= 1) {
		return fmt.Errorf("family %s: a purchase unit needs another unit and a factor over 1", f.Category)
	}
	return nil
}
