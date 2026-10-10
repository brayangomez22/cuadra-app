// Package postgres implements the catalog repositories on PostgreSQL. Every
// operation runs in db.WithTenantTx, so Row-Level Security limits it to one
// tenant even if a query forgets its filter.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// ProductRepository implements domain.ProductRepository.
type ProductRepository struct{ db *db.DB }

// NewProductRepository returns a ProductRepository on d.
func NewProductRepository(d *db.DB) *ProductRepository { return &ProductRepository{db: d} }

// Create stores a new product.
func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
	err := r.db.WithTenantTx(ctx, p.TenantID(), func(tx pgx.Tx) error {
		return sqlcgen.New(tx).CreateProduct(ctx, sqlcgen.CreateProductParams{
			ID:          p.ID(),
			TenantID:    p.TenantID(),
			Sku:         p.SKU(),
			Barcode:     nullable(p.Barcode()),
			Name:        p.Name(),
			Description: p.Description(),
			CategoryID:  p.CategoryID(),
			BaseUnit:    p.BaseUnit().String(),
			Cost:        p.Cost(),
			Price:       p.Price(),
			TaxRate:     p.TaxRate().Rate(),
			Active:      p.IsActive(),
			CreatedAt:   p.CreatedAt(),
		})
	})
	if err := productWriteError(err); err != nil {
		return fmt.Errorf("catalog: create product: %w", err)
	}
	return nil
}

// GetByID returns domain.ErrProductNotFound when the product does not exist
// in tenantID.
func (r *ProductRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	var row sqlcgen.GetProductRow
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		row, err = sqlcgen.New(tx).GetProduct(ctx, id)
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrProductNotFound
	case err != nil:
		return nil, fmt.Errorf("catalog: get product: %w", err)
	}
	return productFromRow(row)
}

// Update stores every field but the id, tenant and creation time.
func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
	var affected int64
	err := r.db.WithTenantTx(ctx, p.TenantID(), func(tx pgx.Tx) error {
		var err error
		affected, err = sqlcgen.New(tx).UpdateProduct(ctx, sqlcgen.UpdateProductParams{
			ID:          p.ID(),
			Sku:         p.SKU(),
			Barcode:     nullable(p.Barcode()),
			Name:        p.Name(),
			Description: p.Description(),
			CategoryID:  p.CategoryID(),
			BaseUnit:    p.BaseUnit().String(),
			Cost:        p.Cost(),
			Price:       p.Price(),
			TaxRate:     p.TaxRate().Rate(),
			Active:      p.IsActive(),
		})
		return err
	})
	if err := productWriteError(err); err != nil {
		return fmt.Errorf("catalog: update product: %w", err)
	}
	if affected == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

// productWriteError translates the constraint violations of a product write.
func productWriteError(err error) error {
	switch {
	case violates(err, uniqueViolation, productsSKUKey):
		return domain.ErrSKUTaken
	case violates(err, uniqueViolation, productsBarcodeKey):
		return domain.ErrBarcodeTaken
	case violates(err, foreignKeyViolation, productsCategoryFKey):
		return domain.ErrCategoryNotFound
	}
	return err
}

// Search returns one page of the tenant's products matching f, and how many
// match in total, read in the same transaction.
func (r *ProductRepository) Search(ctx context.Context, tenantID uuid.UUID, f domain.ProductFilter) (domain.ProductPage, error) {
	var rows []sqlcgen.GetProductRow
	var total int64
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if text := strings.TrimSpace(f.Query); text != "" {
			rows, total, err = search(ctx, q, text, f)
		} else {
			rows, total, err = list(ctx, q, f)
		}
		return err
	})
	if err != nil {
		return domain.ProductPage{}, fmt.Errorf("catalog: search products: %w", err)
	}
	page := domain.ProductPage{Items: make([]*domain.Product, 0, len(rows)), Total: int(total)}
	for _, row := range rows {
		p, err := productFromRow(row)
		if err != nil {
			return domain.ProductPage{}, err
		}
		page.Items = append(page.Items, p)
	}
	return page, nil
}

func list(ctx context.Context, q *sqlcgen.Queries, f domain.ProductFilter) ([]sqlcgen.GetProductRow, int64, error) {
	limit, offset, err := pageBounds(f)
	if err != nil {
		return nil, 0, err
	}
	found, err := q.ListProducts(ctx, sqlcgen.ListProductsParams{
		Active: f.Active, CategoryID: f.CategoryID,
		RowLimit: limit, RowOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountProducts(ctx, sqlcgen.CountProductsParams{Active: f.Active, CategoryID: f.CategoryID})
	if err != nil {
		return nil, 0, err
	}
	rows := make([]sqlcgen.GetProductRow, 0, len(found))
	for _, row := range found {
		rows = append(rows, sqlcgen.GetProductRow(row))
	}
	return rows, total, nil
}

func search(ctx context.Context, q *sqlcgen.Queries, text string, f domain.ProductFilter) ([]sqlcgen.GetProductRow, int64, error) {
	limit, offset, err := pageBounds(f)
	if err != nil {
		return nil, 0, err
	}
	terms := searchTerms(text)
	barcode := text
	found, err := q.SearchProducts(ctx, sqlcgen.SearchProductsParams{
		Active: f.Active, CategoryID: f.CategoryID,
		Sku: strings.ToUpper(text), Barcode: &barcode, Terms: terms, Text: text,
		RowLimit: limit, RowOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountSearchProducts(ctx, sqlcgen.CountSearchProductsParams{
		Active: f.Active, CategoryID: f.CategoryID,
		Sku: strings.ToUpper(text), Barcode: &barcode, Terms: terms,
	})
	if err != nil {
		return nil, 0, err
	}
	rows := make([]sqlcgen.GetProductRow, 0, len(found))
	for _, row := range found {
		rows = append(rows, sqlcgen.GetProductRow(row))
	}
	return rows, total, nil
}

// pageBounds converts the page of f to the query's int32 parameters.
func pageBounds(f domain.ProductFilter) (limit, offset int32, err error) {
	if f.Limit < 0 || f.Limit > math.MaxInt32 || f.Offset < 0 || f.Offset > math.MaxInt32 {
		return 0, 0, fmt.Errorf("catalog: page out of range (limit %d, offset %d)", f.Limit, f.Offset)
	}
	return int32(f.Limit), int32(f.Offset), nil
}

// likeEscaper makes LIKE's wildcards literal: "%" and "_" are searched as text.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// searchTerms splits text into words with LIKE wildcards escaped.
func searchTerms(text string) []string {
	words := strings.Fields(text)
	for i, word := range words {
		words[i] = likeEscaper.Replace(word)
	}
	return words
}

func productFromRow(row sqlcgen.GetProductRow) (*domain.Product, error) {
	unit, err := domain.ParseUnit(row.BaseUnit)
	if err != nil {
		return nil, fmt.Errorf("catalog: product %s: stored unit: %w", row.ID, err)
	}
	rate, err := domain.ParseTaxRate(row.TaxRate)
	if err != nil {
		return nil, fmt.Errorf("catalog: product %s: stored tax rate: %w", row.ID, err)
	}
	var barcode string
	if row.Barcode != nil {
		barcode = *row.Barcode
	}
	p, err := domain.RehydrateProduct(domain.ProductSnapshot{
		ID:          row.ID,
		TenantID:    row.TenantID,
		SKU:         row.Sku,
		Barcode:     barcode,
		Name:        row.Name,
		Description: row.Description,
		CategoryID:  row.CategoryID,
		BaseUnit:    unit,
		Cost:        row.Cost,
		Price:       row.Price,
		TaxRate:     rate,
		Active:      row.Active,
		CreatedAt:   row.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: product %s: %w", row.ID, err)
	}
	return p, nil
}

// nullable stores an empty string as NULL.
func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
