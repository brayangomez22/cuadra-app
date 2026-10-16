package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// SQLSTATEs of the constraint violations the repositories translate.
const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

// Constraints of migrations/00004_catalog.sql and 00010_product_variants.sql
// the adapter maps to domain errors.
const (
	productsCategoryFKey  = "products_category_fkey"
	categoriesParentFKey  = "categories_parent_fkey"
	variantsSKUKey        = "product_variants_tenant_id_sku_key"
	variantsBarcodeKey    = "product_variants_tenant_id_barcode_key"
	variantsAxisValuesKey = "product_variants_axis_values_key"
)

// violates reports whether err violates the constraint named constraint with
// the SQLSTATE code.
func violates(err error, code, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code && pgErr.ConstraintName == constraint
}

// isForeignKeyViolation reports whether err violates any foreign key.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation
}
