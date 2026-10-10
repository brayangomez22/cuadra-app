-- name: CreateProduct :exec
INSERT INTO products (id, tenant_id, sku, barcode, name, description, category_id, base_unit, cost, price, tax_rate, active, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: GetProduct :one
SELECT id, tenant_id, sku, barcode, name, description, category_id, base_unit, cost, price, tax_rate, active, created_at
FROM products
WHERE id = $1;

-- name: UpdateProduct :execrows
UPDATE products
SET sku = $2, barcode = $3, name = $4, description = $5, category_id = $6,
    base_unit = $7, cost = $8, price = $9, tax_rate = $10, active = $11
WHERE id = $1;

-- ListProducts pages the products without a search text, by name.
-- name: ListProducts :many
SELECT id, tenant_id, sku, barcode, name, description, category_id, base_unit, cost, price, tax_rate, active, created_at
FROM products
WHERE (sqlc.narg(active)::boolean IS NULL OR active = sqlc.narg(active))
  AND (sqlc.narg(category_id)::uuid IS NULL OR category_id = sqlc.narg(category_id))
ORDER BY name, id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountProducts :one
SELECT count(*)
FROM products
WHERE (sqlc.narg(active)::boolean IS NULL OR active = sqlc.narg(active))
  AND (sqlc.narg(category_id)::uuid IS NULL OR category_id = sqlc.narg(category_id));

-- SearchProducts matches an exact SKU or barcode, or names containing every
-- term (case- and accent-insensitive). Terms arrive with LIKE wildcards
-- escaped. Exact codes come first, then the names most similar (pg_trgm) to
-- the whole text.
-- name: SearchProducts :many
SELECT id, tenant_id, sku, barcode, name, description, category_id, base_unit, cost, price, tax_rate, active, created_at
FROM products
WHERE (sqlc.narg(active)::boolean IS NULL OR active = sqlc.narg(active))
  AND (sqlc.narg(category_id)::uuid IS NULL OR category_id = sqlc.narg(category_id))
  AND (sku = sqlc.arg(sku)
       OR barcode = sqlc.arg(barcode)
       OR search_name LIKE ALL (SELECT '%' || f_unaccent(lower(term)) || '%' FROM unnest(sqlc.arg(terms)::text[]) AS term))
ORDER BY coalesce(sku = sqlc.arg(sku) OR barcode = sqlc.arg(barcode), false) DESC,
         similarity(search_name, f_unaccent(lower(sqlc.arg(text)))) DESC,
         name, id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountSearchProducts :one
SELECT count(*)
FROM products
WHERE (sqlc.narg(active)::boolean IS NULL OR active = sqlc.narg(active))
  AND (sqlc.narg(category_id)::uuid IS NULL OR category_id = sqlc.narg(category_id))
  AND (sku = sqlc.arg(sku)
       OR barcode = sqlc.arg(barcode)
       OR search_name LIKE ALL (SELECT '%' || f_unaccent(lower(term)) || '%' FROM unnest(sqlc.arg(terms)::text[]) AS term));
