package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// Search returns one page of the tenant's products matching f, and how many
// match in total, read in the same transaction.
func (r *ProductRepository) Search(ctx context.Context, tenantID uuid.UUID, f domain.ProductFilter) (domain.ProductPage, error) {
	var rows []sqlcgen.GetProductRow
	var variants []sqlcgen.ListVariantsRow
	var units []sqlcgen.ListVariantUnitsRow
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
		if variants, err = q.ListVariants(ctx, ids); err != nil {
			return err
		}
		units, err = q.ListVariantUnits(ctx, variantIDs(variants))
		return err
	})
	if err != nil {
		return domain.ProductPage{}, fmt.Errorf("catalog: search products: %w", err)
	}
	byProduct := make(map[uuid.UUID][]sqlcgen.ListVariantsRow, len(rows))
	for _, v := range variants {
		byProduct[v.ProductID] = append(byProduct[v.ProductID], v)
	}
	byVariant := unitsByVariant(units)
	page := domain.ProductPage{Items: make([]*domain.Product, 0, len(rows)), Total: int(total)}
	for _, row := range rows {
		p, err := productFromRows(row, byProduct[row.ID], byVariant)
		if err != nil {
			return domain.ProductPage{}, err
		}
		page.Items = append(page.Items, p)
	}
	return page, nil
}

func list(ctx context.Context, q *sqlcgen.Queries, f domain.ProductFilter) ([]sqlcgen.GetProductRow, int64, error) {
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
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
	limit, offset, err := db.PageBounds(f.Limit, f.Offset)
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
