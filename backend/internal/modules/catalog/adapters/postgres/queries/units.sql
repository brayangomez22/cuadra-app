-- name: CreateUnit :exec
INSERT INTO units_of_measure (id, tenant_id, code, name, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetUnit :one
SELECT id, tenant_id, code, name, created_at
FROM units_of_measure
WHERE id = $1;

-- name: GetUnitByCode :one
SELECT id, tenant_id, code, name, created_at
FROM units_of_measure
WHERE code = $1;

-- name: ListUnits :many
SELECT id, tenant_id, code, name, created_at
FROM units_of_measure
ORDER BY name, id;

-- name: UpdateUnit :execrows
UPDATE units_of_measure
SET code = $2, name = $3
WHERE id = $1;

-- name: DeleteUnit :execrows
DELETE FROM units_of_measure
WHERE id = $1;
