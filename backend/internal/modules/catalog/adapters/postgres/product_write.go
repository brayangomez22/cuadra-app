package postgres

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

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
			return codeTaken(err, v.SKU(), v.Barcode())
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
	if err := createVariantUnits(ctx, q, p); err != nil {
		return err
	}
	return checkBarcodesElsewhere(ctx, q, p)
}

// createVariantUnits stores the alternate units of every variant of p.
func createVariantUnits(ctx context.Context, q *sqlcgen.Queries, p *domain.Product) error {
	for _, v := range p.Variants() {
		for _, u := range v.Units() {
			if err := q.CreateVariantUnit(ctx, sqlcgen.CreateVariantUnitParams{
				TenantID: p.TenantID(), VariantID: v.ID(), UnitID: u.Unit().ID,
				Factor: u.Factor(), Price: u.Price(), Barcode: nullable(u.Barcode()),
			}); err != nil {
				return codeTaken(err, "", u.Barcode())
			}
		}
	}
	return nil
}

// checkBarcodesElsewhere fails with domain.ErrBarcodeTaken, naming the
// barcode, when a barcode of p's variants or units is taken in the other
// table by another product.
func checkBarcodesElsewhere(ctx context.Context, q *sqlcgen.Queries, p *domain.Product) error {
	variantCodes, unitCodes := barcodesOf(p)
	if len(variantCodes) == 0 && len(unitCodes) == 0 {
		return nil
	}
	taken, err := q.BarcodesTakenElsewhere(ctx, sqlcgen.BarcodesTakenElsewhereParams{
		ProductID: p.ID(), VariantBarcodes: variantCodes, UnitBarcodes: unitCodes,
	})
	if err != nil {
		return err
	}
	if len(taken) > 0 {
		return &domain.CodeTakenError{Code: taken[0], Err: domain.ErrBarcodeTaken}
	}
	return nil
}

// barcodesOf returns the barcodes of p's variants and of their units.
func barcodesOf(p *domain.Product) (variantCodes, unitCodes []string) {
	for _, v := range p.Variants() {
		if v.Barcode() != "" {
			variantCodes = append(variantCodes, v.Barcode())
		}
		for _, u := range v.Units() {
			if u.Barcode() != "" {
				unitCodes = append(unitCodes, u.Barcode())
			}
		}
	}
	return variantCodes, unitCodes
}

// codeTaken names the code of a unique violation on a variant's SKU or on
// a variant's or a unit's barcode; any other error comes back as it is.
func codeTaken(err error, sku, barcode string) error {
	switch {
	case db.Violates(err, db.UniqueViolation, variantsSKUKey):
		return &domain.CodeTakenError{Code: sku, Err: domain.ErrSKUTaken}
	case db.Violates(err, db.UniqueViolation, variantsBarcodeKey), db.Violates(err, db.UniqueViolation, variantUnitsBarcodeKey):
		return &domain.CodeTakenError{Code: barcode, Err: domain.ErrBarcodeTaken}
	}
	return err
}

// productWriteError translates the constraint violations of a product write.
func productWriteError(err error) error {
	switch {
	case db.Violates(err, db.UniqueViolation, variantsSKUKey):
		return domain.ErrSKUTaken
	case db.Violates(err, db.UniqueViolation, variantsBarcodeKey), db.Violates(err, db.UniqueViolation, variantUnitsBarcodeKey):
		return domain.ErrBarcodeTaken
	case db.Violates(err, db.UniqueViolation, variantsAxisValuesKey):
		return domain.ErrDuplicateVariant
	case db.Violates(err, db.ForeignKeyViolation, productsCategoryFKey):
		return domain.ErrCategoryNotFound
	case db.Violates(err, db.ForeignKeyViolation, productsBaseUnitFKey), db.Violates(err, db.ForeignKeyViolation, variantUnitsUnitFKey):
		return domain.ErrUnitNotFound
	}
	return err
}
