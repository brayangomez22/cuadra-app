// Package postgres implements the catalog repositories on PostgreSQL. Every
// operation runs in db.WithTenantTx, so Row-Level Security limits it to one
// tenant even if a query forgets its filter.
package postgres

import (
	"context"
	"encoding/json"
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

// Create stores a new product and its variants in one transaction.
func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
	err := r.db.WithTenantTx(ctx, p.TenantID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := q.CreateProduct(ctx, sqlcgen.CreateProductParams{
			ID:          p.ID(),
			TenantID:    p.TenantID(),
			Name:        p.Name(),
			Description: p.Description(),
			CategoryID:  p.CategoryID(),
			BaseUnitID:  p.BaseUnit().ID,
			TaxRate:     p.TaxRate().Rate(),
			Active:      p.IsActive(),
			CreatedAt:   p.CreatedAt(),
		}); err != nil {
			return err
		}
		return saveVariants(ctx, q, p)
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
	var variants []sqlcgen.ListVariantsRow
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if row, err = q.GetProduct(ctx, id); err != nil {
			return err
		}
		variants, err = q.ListVariants(ctx, []uuid.UUID{id})
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrProductNotFound
	case err != nil:
		return nil, fmt.Errorf("catalog: get product: %w", err)
	}
	return productFromRows(row, variants)
}

// Update stores every field of the product and of its variants but their
// ids, tenant and creation time, and inserts the variants it does not have
// yet. Variants are never deleted: they are deactivated.
func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
	err := r.db.WithTenantTx(ctx, p.TenantID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		affected, err := q.UpdateProduct(ctx, sqlcgen.UpdateProductParams{
			ID:          p.ID(),
			Name:        p.Name(),
			Description: p.Description(),
			CategoryID:  p.CategoryID(),
			BaseUnitID:  p.BaseUnit().ID,
			TaxRate:     p.TaxRate().Rate(),
			Active:      p.IsActive(),
		})
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrProductNotFound
		}
		return saveVariants(ctx, q, p)
	})
	if errors.Is(err, domain.ErrProductNotFound) {
		return err
	}
	if err := productWriteError(err); err != nil {
		return fmt.Errorf("catalog: update product: %w", err)
	}
	return nil
}

// saveVariants inserts or updates each variant of p. Updating one that
// belongs to another product changes no row and is reported as
// domain.ErrVariantNotFound.
func saveVariants(ctx context.Context, q *sqlcgen.Queries, p *domain.Product) error {
	for _, v := range p.Variants() {
		axes, err := json.Marshal(v.AxisValues())
		if err != nil {
			return err
		}
		affected, err := q.SaveVariant(ctx, sqlcgen.SaveVariantParams{
			ID:         v.ID(),
			TenantID:   p.TenantID(),
			ProductID:  p.ID(),
			Sku:        v.SKU(),
			Barcode:    nullable(v.Barcode()),
			Cost:       v.Cost(),
			Price:      v.Price(),
			AxisValues: axes,
			IsDefault:  v.IsDefault(),
			Active:     v.IsActive(),
			CreatedAt:  v.CreatedAt(),
		})
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrVariantNotFound
		}
	}
	return nil
}

// productWriteError translates the constraint violations of a product write.
func productWriteError(err error) error {
	switch {
	case violates(err, uniqueViolation, variantsSKUKey):
		return domain.ErrSKUTaken
	case violates(err, uniqueViolation, variantsBarcodeKey):
		return domain.ErrBarcodeTaken
	case violates(err, uniqueViolation, variantsAxisValuesKey):
		return domain.ErrDuplicateVariant
	case violates(err, foreignKeyViolation, productsCategoryFKey):
		return domain.ErrCategoryNotFound
	case violates(err, foreignKeyViolation, productsBaseUnitFKey):
		return domain.ErrUnitNotFound
	}
	return err
}

// Search returns one page of the tenant's products matching f, and how many
// match in total, read in the same transaction.
func (r *ProductRepository) Search(ctx context.Context, tenantID uuid.UUID, f domain.ProductFilter) (domain.ProductPage, error) {
	var rows []sqlcgen.GetProductRow
	var variants []sqlcgen.ListVariantsRow
	var total int64
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if text := strings.TrimSpace(f.Query); text != "" {
			rows, total, err = search(ctx, q, text, f)
		} else {
			rows, total, err = list(ctx, q, f)
		}
		if err != nil || len(rows) == 0 {
			return err
		}
		ids := make([]uuid.UUID, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
		variants, err = q.ListVariants(ctx, ids)
		return err
	})
	if err != nil {
		return domain.ProductPage{}, fmt.Errorf("catalog: search products: %w", err)
	}
	byProduct := make(map[uuid.UUID][]sqlcgen.ListVariantsRow, len(rows))
	for _, v := range variants {
		byProduct[v.ProductID] = append(byProduct[v.ProductID], v)
	}
	page := domain.ProductPage{Items: make([]*domain.Product, 0, len(rows)), Total: int(total)}
	for _, row := range rows {
		p, err := productFromRows(row, byProduct[row.ID])
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
		Active: f.Active, CategoryID: f.CategoryID, Ids: f.IDs,
		RowLimit: limit, RowOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountProducts(ctx, sqlcgen.CountProductsParams{Active: f.Active, CategoryID: f.CategoryID, Ids: f.IDs})
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
		Active: f.Active, CategoryID: f.CategoryID, Ids: f.IDs,
		Sku: strings.ToUpper(text), Barcode: &barcode, Terms: terms, Text: text,
		RowLimit: limit, RowOffset: offset,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := q.CountSearchProducts(ctx, sqlcgen.CountSearchProductsParams{
		Active: f.Active, CategoryID: f.CategoryID, Ids: f.IDs,
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

// productFromRows rebuilds a product from its row and its variants' rows.
func productFromRows(row sqlcgen.GetProductRow, variantRows []sqlcgen.ListVariantsRow) (*domain.Product, error) {
	unit := domain.UnitRef{ID: row.BaseUnitID, Code: row.UnitCode, Name: row.UnitName}
	rate, err := domain.ParseTaxRate(row.TaxRate)
	if err != nil {
		return nil, fmt.Errorf("catalog: product %s: stored tax rate: %w", row.ID, err)
	}
	variants := make([]domain.VariantSnapshot, 0, len(variantRows))
	for _, v := range variantRows {
		var axes map[string]string
		if err := json.Unmarshal(v.AxisValues, &axes); err != nil {
			return nil, fmt.Errorf("catalog: variant %s: stored axis values: %w", v.ID, err)
		}
		var barcode string
		if v.Barcode != nil {
			barcode = *v.Barcode
		}
		variants = append(variants, domain.VariantSnapshot{
			ID:         v.ID,
			SKU:        v.Sku,
			Barcode:    barcode,
			Cost:       v.Cost,
			Price:      v.Price,
			AxisValues: axes,
			IsDefault:  v.IsDefault,
			Active:     v.Active,
			CreatedAt:  v.CreatedAt,
		})
	}
	p, err := domain.RehydrateProduct(domain.ProductSnapshot{
		ID:          row.ID,
		TenantID:    row.TenantID,
		Name:        row.Name,
		Description: row.Description,
		CategoryID:  row.CategoryID,
		BaseUnit:    unit,
		TaxRate:     rate,
		Active:      row.Active,
		CreatedAt:   row.CreatedAt,
		Variants:    variants,
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

// VariantProducts returns the product of each of variantIDs that exists in
// tenantID; the others are left out.
func (r *ProductRepository) VariantProducts(ctx context.Context, tenantID uuid.UUID, variantIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	var rows []sqlcgen.VariantProductsRow
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		rows, err = sqlcgen.New(tx).VariantProducts(ctx, variantIDs)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("catalog: products of variants: %w", err)
	}
	out := make(map[uuid.UUID]uuid.UUID, len(rows))
	for _, row := range rows {
		out[row.ID] = row.ProductID
	}
	return out, nil
}
