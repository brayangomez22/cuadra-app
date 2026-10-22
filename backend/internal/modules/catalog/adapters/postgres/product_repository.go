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
	attrs, err := attributesJSON(p.Attributes())
	if err != nil {
		return err
	}
	err = r.db.WithTenantTx(ctx, p.TenantID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := q.CreateProduct(ctx, sqlcgen.CreateProductParams{
			ID:          p.ID(),
			TenantID:    p.TenantID(),
			Name:        p.Name(),
			Description: p.Description(),
			CategoryID:  p.CategoryID(),
			BaseUnitID:  p.BaseUnit().ID,
			TaxRate:     p.TaxRate().Rate(),
			Attributes:  attrs,
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
	var units []sqlcgen.ListVariantUnitsRow
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		var err error
		if row, err = q.GetProduct(ctx, id); err != nil {
			return err
		}
		if variants, err = q.ListVariants(ctx, []uuid.UUID{id}); err != nil {
			return err
		}
		units, err = q.ListVariantUnits(ctx, variantIDs(variants))
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrProductNotFound
	case err != nil:
		return nil, fmt.Errorf("catalog: get product: %w", err)
	}
	return productFromRows(row, variants, unitsByVariant(units))
}

// Update stores every field of the product and of its variants but their
// ids, tenant and creation time, and inserts the variants it does not have
// yet. Variants are never deleted: they are deactivated.
func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
	attrs, err := attributesJSON(p.Attributes())
	if err != nil {
		return err
	}
	err = r.db.WithTenantTx(ctx, p.TenantID(), func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		affected, err := q.UpdateProduct(ctx, sqlcgen.UpdateProductParams{
			ID:          p.ID(),
			Name:        p.Name(),
			Description: p.Description(),
			CategoryID:  p.CategoryID(),
			BaseUnitID:  p.BaseUnit().ID,
			TaxRate:     p.TaxRate().Rate(),
			Attributes:  attrs,
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
	if errors.Is(err, domain.ErrProductNotFound) || errors.Is(err, domain.ErrBarcodeTaken) {
		return err
	}
	if err := productWriteError(err); err != nil {
		return fmt.Errorf("catalog: update product: %w", err)
	}
	return nil
}

// saveVariants inserts or updates each variant of p and stores again their
// alternate units. Updating a variant that belongs to another product
// changes no row and is reported as domain.ErrVariantNotFound.
func saveVariants(ctx context.Context, q *sqlcgen.Queries, p *domain.Product) error {
	ids := make([]uuid.UUID, 0, len(p.Variants()))
	for _, v := range p.Variants() {
		ids = append(ids, v.ID())
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
	return saveUnits(ctx, q, p, ids)
}

// saveUnits replaces the alternate units of the variants ids of p with the
// ones p has, and checks their barcodes against the other table: a unique
// key cannot span product_variants and variant_units.
func saveUnits(ctx context.Context, q *sqlcgen.Queries, p *domain.Product, ids []uuid.UUID) error {
	if err := q.DeleteVariantUnits(ctx, ids); err != nil {
		return err
	}
	var variantCodes, unitCodes []string
	for _, v := range p.Variants() {
		if v.Barcode() != "" {
			variantCodes = append(variantCodes, v.Barcode())
		}
		for _, u := range v.Units() {
			if err := q.CreateVariantUnit(ctx, sqlcgen.CreateVariantUnitParams{
				TenantID: p.TenantID(), VariantID: v.ID(), UnitID: u.Unit().ID,
				Factor: u.Factor(), Price: u.Price(), Barcode: nullable(u.Barcode()),
			}); err != nil {
				return err
			}
			if u.Barcode() != "" {
				unitCodes = append(unitCodes, u.Barcode())
			}
		}
	}
	if len(variantCodes) == 0 && len(unitCodes) == 0 {
		return nil
	}
	taken, err := q.BarcodeTakenElsewhere(ctx, sqlcgen.BarcodeTakenElsewhereParams{
		ProductID: p.ID(), VariantBarcodes: variantCodes, UnitBarcodes: unitCodes,
	})
	if err != nil {
		return err
	}
	if taken != nil && *taken {
		return domain.ErrBarcodeTaken
	}
	return nil
}

// productWriteError translates the constraint violations of a product write.
func productWriteError(err error) error {
	switch {
	case violates(err, uniqueViolation, variantsSKUKey):
		return domain.ErrSKUTaken
	case violates(err, uniqueViolation, variantsBarcodeKey), violates(err, uniqueViolation, variantUnitsBarcodeKey):
		return domain.ErrBarcodeTaken
	case violates(err, uniqueViolation, variantsAxisValuesKey):
		return domain.ErrDuplicateVariant
	case violates(err, foreignKeyViolation, productsCategoryFKey):
		return domain.ErrCategoryNotFound
	case violates(err, foreignKeyViolation, productsBaseUnitFKey), violates(err, foreignKeyViolation, variantUnitsUnitFKey):
		return domain.ErrUnitNotFound
	}
	return err
}

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

// variantIDs returns the ids of the variants' rows.
func variantIDs(rows []sqlcgen.ListVariantsRow) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(rows))
	for _, v := range rows {
		ids = append(ids, v.ID)
	}
	return ids
}

// unitsByVariant groups the alternate units' rows by variant.
func unitsByVariant(rows []sqlcgen.ListVariantUnitsRow) map[uuid.UUID][]domain.VariantUnitParams {
	out := map[uuid.UUID][]domain.VariantUnitParams{}
	for _, u := range rows {
		var barcode string
		if u.Barcode != nil {
			barcode = *u.Barcode
		}
		out[u.VariantID] = append(out[u.VariantID], domain.VariantUnitParams{
			Unit:   domain.UnitRef{ID: u.UnitID, Code: u.UnitCode, Name: u.UnitName},
			Factor: u.Factor, Price: u.Price, Barcode: barcode,
		})
	}
	return out
}

// productFromRows rebuilds a product from its row, its variants' rows and
// the alternate units of its variants.
func productFromRows(row sqlcgen.GetProductRow, variantRows []sqlcgen.ListVariantsRow, units map[uuid.UUID][]domain.VariantUnitParams) (*domain.Product, error) {
	unit := domain.UnitRef{ID: row.BaseUnitID, Code: row.UnitCode, Name: row.UnitName}
	rate, err := domain.ParseTaxRate(row.TaxRate)
	if err != nil {
		return nil, fmt.Errorf("catalog: product %s: stored tax rate: %w", row.ID, err)
	}
	attrs, err := attributesFromJSON(row.Attributes)
	if err != nil {
		return nil, fmt.Errorf("catalog: product %s: stored attributes: %w", row.ID, err)
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
			Units:      units[v.ID],
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
		Attributes:  attrs,
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

// attributesJSON encodes a product's attributes as the JSON object stored
// in products.attributes: a string for text, number and list values, a
// boolean for yes/no ones.
func attributesJSON(a domain.Attributes) ([]byte, error) {
	values := a.Values()
	out := make(map[string]any, len(values))
	for key, v := range values {
		if v.IsBool() {
			out[key] = v.Bool()
		} else {
			out[key] = v.Text()
		}
	}
	return json.Marshal(out)
}

// attributesFromJSON decodes products.attributes.
func attributesFromJSON(raw []byte) (domain.Attributes, error) {
	var stored map[string]any
	if err := json.Unmarshal(raw, &stored); err != nil {
		return domain.Attributes{}, err
	}
	values := make(map[string]domain.AttributeValue, len(stored))
	for key, v := range stored {
		switch v := v.(type) {
		case string:
			values[key] = domain.TextValue(v)
		case bool:
			values[key] = domain.BoolValue(v)
		default:
			return domain.Attributes{}, fmt.Errorf("attribute %q holds a %T", key, v)
		}
	}
	return domain.RehydrateAttributes(values)
}
