-- ListVariants returns the variants of the given products: the default one
-- first, then in the order they were added (UUID v7 ids follow creation).
-- name: ListVariants :many
SELECT id, product_id, sku, barcode, cost, price, axis_values, is_default, active, created_at
FROM product_variants
WHERE product_id = ANY(sqlc.arg(product_ids)::uuid[])
ORDER BY product_id, is_default DESC, created_at, id;

-- SaveVariant inserts a variant or updates every editable field of an
-- existing one. The WHERE keeps an update inside its product.
-- name: SaveVariant :execrows
INSERT INTO product_variants (id, tenant_id, product_id, sku, barcode, cost, price, axis_values, is_default, active, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id) DO UPDATE
SET sku = EXCLUDED.sku, barcode = EXCLUDED.barcode, cost = EXCLUDED.cost, price = EXCLUDED.price,
    axis_values = EXCLUDED.axis_values, active = EXCLUDED.active
WHERE product_variants.product_id = EXCLUDED.product_id;

-- VariantProducts returns the product of each of the given variants, for
-- other modules that only know the variant.
-- name: VariantProducts :many
SELECT id, product_id
FROM product_variants
WHERE id = ANY(sqlc.arg(variant_ids)::uuid[]);

-- ListVariantUnits returns the alternate units of the given variants with
-- the code and name of each unit.
-- name: ListVariantUnits :many
SELECT vu.variant_id, vu.unit_id, u.code AS unit_code, u.name AS unit_name, vu.factor, vu.price, vu.barcode
FROM variant_units vu
JOIN units_of_measure u ON u.tenant_id = vu.tenant_id AND u.id = vu.unit_id
WHERE vu.variant_id = ANY(sqlc.arg(variant_ids)::uuid[])
ORDER BY vu.variant_id, vu.factor, u.code;

-- DeleteVariantUnits removes the alternate units of the given variants, so
-- a product write stores them again as the product has them.
-- name: DeleteVariantUnits :exec
DELETE FROM variant_units
WHERE variant_id = ANY(sqlc.arg(variant_ids)::uuid[]);

-- name: CreateVariantUnit :exec
INSERT INTO variant_units (tenant_id, variant_id, unit_id, factor, price, barcode)
VALUES ($1, $2, $3, $4, $5, $6);

-- BarcodeTakenElsewhere reports whether a barcode of a product is used in
-- the other table by another product of the tenant: one of its variants'
-- by an alternate unit, or one of its units' by a variant. Each table's
-- unique key covers the rest.
-- name: BarcodeTakenElsewhere :one
SELECT EXISTS (
           SELECT 1
           FROM variant_units vu
           JOIN product_variants v ON v.tenant_id = vu.tenant_id AND v.id = vu.variant_id
           WHERE v.product_id <> sqlc.arg(product_id) AND vu.barcode = ANY(sqlc.arg(variant_barcodes)::text[]))
    OR EXISTS (
           SELECT 1
           FROM product_variants v
           WHERE v.product_id <> sqlc.arg(product_id) AND v.barcode = ANY(sqlc.arg(unit_barcodes)::text[]));

-- SKUProducts returns the products with a variant whose SKU is among the
-- given ones (SKUs are stored trimmed and in uppercase).
-- name: SKUProducts :many
SELECT DISTINCT product_id
FROM product_variants
WHERE sku = ANY(sqlc.arg(skus)::text[]);
