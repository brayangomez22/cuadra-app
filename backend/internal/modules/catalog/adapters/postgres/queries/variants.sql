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
