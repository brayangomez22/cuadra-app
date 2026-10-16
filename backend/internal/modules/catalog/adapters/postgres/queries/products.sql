-- name: CreateProduct :exec
INSERT INTO products (id, tenant_id, name, description, category_id, base_unit, tax_rate, active, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetProduct :one
SELECT id, tenant_id, name, description, category_id, base_unit, tax_rate, active, created_at
FROM products
WHERE id = $1;

-- name: UpdateProduct :execrows
UPDATE products
SET name = $2, description = $3, category_id = $4, base_unit = $5, tax_rate = $6, active = $7
WHERE id = $1;

-- ListProducts pages the products without a search text, by name.
-- name: ListProducts :many
SELECT id, tenant_id, name, description, category_id, base_unit, tax_rate, active, created_at
FROM products
WHERE (sqlc.narg(active)::boolean IS NULL OR active = sqlc.narg(active))
  AND (sqlc.narg(category_id)::uuid IS NULL OR category_id = sqlc.narg(category_id))
  AND (sqlc.narg(ids)::uuid[] IS NULL OR id = ANY(sqlc.narg(ids)::uuid[]))
ORDER BY name, id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountProducts :one
SELECT count(*)
FROM products
WHERE (sqlc.narg(active)::boolean IS NULL OR active = sqlc.narg(active))
  AND (sqlc.narg(category_id)::uuid IS NULL OR category_id = sqlc.narg(category_id))
  AND (sqlc.narg(ids)::uuid[] IS NULL OR id = ANY(sqlc.narg(ids)::uuid[]));

-- SearchProducts matches the exact SKU or barcode of any of a product's
-- variants, or names containing every term (case- and accent-insensitive).
-- Terms arrive with LIKE wildcards escaped. Exact codes come first, then the
-- names most similar (pg_trgm) to the whole text.
-- name: SearchProducts :many
SELECT p.id, p.tenant_id, p.name, p.description, p.category_id, p.base_unit, p.tax_rate, p.active, p.created_at
FROM products p
CROSS JOIN LATERAL (
    SELECT EXISTS (
        SELECT 1 FROM product_variants v
        WHERE v.product_id = p.id AND (v.sku = sqlc.arg(sku) OR v.barcode = sqlc.arg(barcode))
    ) AS by_code
) code
WHERE (sqlc.narg(active)::boolean IS NULL OR p.active = sqlc.narg(active))
  AND (sqlc.narg(category_id)::uuid IS NULL OR p.category_id = sqlc.narg(category_id))
  AND (sqlc.narg(ids)::uuid[] IS NULL OR p.id = ANY(sqlc.narg(ids)::uuid[]))
  AND (code.by_code
       OR p.search_name LIKE ALL (SELECT '%' || f_unaccent(lower(term)) || '%' FROM unnest(sqlc.arg(terms)::text[]) AS term))
ORDER BY code.by_code DESC,
         similarity(p.search_name, f_unaccent(lower(sqlc.arg(text)))) DESC,
         p.name, p.id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountSearchProducts :one
SELECT count(*)
FROM products p
WHERE (sqlc.narg(active)::boolean IS NULL OR p.active = sqlc.narg(active))
  AND (sqlc.narg(category_id)::uuid IS NULL OR p.category_id = sqlc.narg(category_id))
  AND (sqlc.narg(ids)::uuid[] IS NULL OR p.id = ANY(sqlc.narg(ids)::uuid[]))
  AND (EXISTS (
           SELECT 1 FROM product_variants v
           WHERE v.product_id = p.id AND (v.sku = sqlc.arg(sku) OR v.barcode = sqlc.arg(barcode)))
       OR p.search_name LIKE ALL (SELECT '%' || f_unaccent(lower(term)) || '%' FROM unnest(sqlc.arg(terms)::text[]) AS term));
