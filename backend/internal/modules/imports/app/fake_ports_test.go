package app_test

import (
	"bytes"
	"context"
	"errors"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// testAxes are the variant axes of the tenant in the tests.
var testAxes = []domain.AxisColumn{{Name: "talla", Values: []string{"S", "M", "L"}}}

// errConnection is an infrastructure failure: the job is retried.
var errConnection = errors.New("connection reset")

// fakeCatalog keeps the variants by SKU. A SKU starting with "MALO" is
// rejected as the catalog would reject an invalid one.
type fakeCatalog struct {
	mu sync.Mutex
	// products maps each SKU to the name of its product.
	products map[string]string
	ids      map[string]uuid.UUID
	// failOnCall makes the nth call to ImportProducts fail once with
	// errConnection (0 never).
	failOnCall, calls int
	// failSKU is rejected only when it is written, as a barcode another
	// product took after the validation would be.
	failSKU string
}

func newFakeCatalog() *fakeCatalog {
	return &fakeCatalog{products: map[string]string{}, ids: map[string]uuid.UUID{}}
}

func (c *fakeCatalog) Layout(context.Context, app.Actor) (domain.Layout, error) {
	return domain.Layout{Units: []string{"und"}, Axes: testAxes, DefaultTaxRate: dec("0.19")}, nil
}

func (c *fakeCatalog) CheckProducts(_ context.Context, _ app.Actor, groups []domain.Group, p domain.ExistingPolicy) ([]app.ProductResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]app.ProductResult, 0, len(groups))
	for _, g := range groups {
		out = append(out, c.plan(g, p, false))
	}
	return out, nil
}

func (c *fakeCatalog) ImportProducts(_ context.Context, _ app.Actor, groups []domain.Group, p domain.ExistingPolicy) ([]app.ProductResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if c.calls == c.failOnCall {
		return nil, errConnection
	}
	out := make([]app.ProductResult, 0, len(groups))
	for _, g := range groups {
		r := c.plan(g, p, true)
		if slices.ContainsFunc(g.Variants, func(v domain.VariantLine) bool { return v.SKU == c.failSKU }) {
			r = app.ProductResult{Problem: &app.Problem{Code: "barcode_taken", Message: "Ya existe ese código de barras."}}
		}
		out = append(out, r)
		if r.Problem != nil {
			break
		}
	}
	return out, nil
}

// plan decides what happens to g and, when save is true, stores it.
func (c *fakeCatalog) plan(g domain.Group, p domain.ExistingPolicy, save bool) app.ProductResult {
	r := app.ProductResult{Outcome: app.OutcomeCreated}
	for _, v := range g.Variants {
		if strings.HasPrefix(v.SKU, "MALO") {
			return app.ProductResult{Problem: &app.Problem{Code: "invalid_sku", Message: "El SKU no es válido."}}
		}
		if _, ok := c.products[v.SKU]; ok {
			r.Outcome = app.OutcomeUpdated
		}
	}
	if r.Outcome == app.OutcomeUpdated && p == domain.SkipExisting {
		return app.ProductResult{Outcome: app.OutcomeSkipped}
	}
	for _, v := range g.Variants {
		id, existed := c.ids[v.SKU]
		if !existed {
			id = uuid.Must(uuid.NewV7())
		}
		r.Variants = append(r.Variants, app.VariantResult{ID: id, Created: !existed})
		if save {
			c.products[v.SKU], c.ids[v.SKU] = g.Product.Name, id
		}
	}
	return r
}

func (c *fakeCatalog) snapshot() (map[string]string, map[string]uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return maps.Clone(c.products), maps.Clone(c.ids)
}

func (c *fakeCatalog) restore(products map[string]string, ids map[string]uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.products, c.ids = products, ids
}

// fakeStock keeps the initial stock entries it received.
type fakeStock struct {
	mu        sync.Mutex
	locations map[uuid.UUID]bool
	entries   []app.StockEntry
}

func (s *fakeStock) RequireLocation(_ context.Context, _ app.Actor, id uuid.UUID) error {
	if !s.locations[id] {
		return app.ErrLocationNotFound
	}
	return nil
}

func (s *fakeStock) ReceiveInitial(_ context.Context, _ app.Actor, _ uuid.UUID, entries []app.StockEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entries...)
	return nil
}

type fakeModules struct{ active map[string]bool }

func (m *fakeModules) IsActive(_ context.Context, _ uuid.UUID, id string) (bool, error) {
	return m.active[id], nil
}

// fakeSheets reads files of lines with cells separated by "|", and writes a
// template as its headers joined the same way.
type fakeSheets struct{}

func (fakeSheets) Rows(_ domain.Format, data []byte) (app.Sheet, error) {
	rows := func(yield func([]string, error) bool) {
		for line := range bytes.Lines(data) {
			if !yield(strings.Split(strings.TrimRight(string(line), "\n"), "|"), nil) {
				return
			}
		}
	}
	return app.Sheet{Rows: iter.Seq2[[]string, error](rows), Style: domain.PlainNumbers}, nil
}

func (fakeSheets) Template(_ domain.Format, l domain.Layout) ([]byte, error) {
	return []byte(strings.Join(l.Headers(), "|")), nil
}

// fakeTx runs fn and, if it fails, puts the catalog, the stock and the
// imports back as they were, as a rolled back transaction would.
type fakeTx struct {
	catalog *fakeCatalog
	stock   *fakeStock
	imports *fakeImports
}

func (tx fakeTx) InTx(ctx context.Context, _ uuid.UUID, fn func(context.Context) error) error {
	products, ids := tx.catalog.snapshot()
	tx.stock.mu.Lock()
	entries := slices.Clone(tx.stock.entries)
	tx.stock.mu.Unlock()
	saved := tx.imports.snapshot()
	if err := fn(ctx); err != nil {
		tx.catalog.restore(products, ids)
		tx.stock.mu.Lock()
		tx.stock.entries = entries
		tx.stock.mu.Unlock()
		tx.imports.restore(saved)
		return err
	}
	return nil
}
