package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

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
