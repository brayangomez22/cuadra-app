-- name: CreateVariantAxis :exec
INSERT INTO variant_axes (id, tenant_id, name, allowed_values, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetVariantAxis :one
SELECT id, tenant_id, name, allowed_values, created_at
FROM variant_axes
WHERE id = $1;

-- name: ListVariantAxes :many
SELECT id, tenant_id, name, allowed_values, created_at
FROM variant_axes
ORDER BY name, id;

-- UpdateVariantAxis changes the allowed values; the name never changes.
-- name: UpdateVariantAxis :execrows
UPDATE variant_axes
SET allowed_values = $2
WHERE id = $1;

-- name: DeleteVariantAxis :one
DELETE FROM variant_axes
WHERE id = $1
RETURNING name;

-- VariantAxisInUse reports whether a variant has a value on the axis.
-- name: VariantAxisInUse :one
SELECT EXISTS (
    SELECT 1 FROM product_variants WHERE axis_values ? sqlc.arg(name)::text
);

-- VariantAxisValuesInUse reports whether a variant has one of the values on
-- the axis, regardless of case: a variant stored before its tenant defined
-- the axis may spell a value differently.
-- name: VariantAxisValuesInUse :one
SELECT EXISTS (
    SELECT 1 FROM product_variants
    WHERE lower(axis_values ->> sqlc.arg(name)::text) IN (
        SELECT lower(value) FROM unnest(sqlc.arg(axis_values)::text[]) AS value
    )
);
