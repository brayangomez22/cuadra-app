-- name: CreateAttribute :exec
INSERT INTO attribute_definitions (id, tenant_id, key, label, type, options, required, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetAttribute :one
SELECT id, tenant_id, key, label, type, options, required, created_at
FROM attribute_definitions
WHERE id = $1;

-- name: ListAttributes :many
SELECT id, tenant_id, key, label, type, options, required, created_at
FROM attribute_definitions
ORDER BY label, id;

-- UpdateAttribute changes what may change; the key and the type never do.
-- name: UpdateAttribute :execrows
UPDATE attribute_definitions
SET label = $2, options = $3, required = $4
WHERE id = $1;

-- name: DeleteAttribute :one
DELETE FROM attribute_definitions
WHERE id = $1
RETURNING key;

-- AttributeInUse reports whether a product has a value for the key.
-- name: AttributeInUse :one
SELECT EXISTS (
    SELECT 1 FROM products WHERE attributes ? sqlc.arg(key)::text
);

-- AttributeOptionsInUse reports whether a product has one of the options
-- as its value for the key. Products store an option as its definition
-- spells it.
-- name: AttributeOptionsInUse :one
SELECT EXISTS (
    SELECT 1 FROM products
    WHERE attributes ->> sqlc.arg(key)::text = ANY(sqlc.arg(options)::text[])
);
