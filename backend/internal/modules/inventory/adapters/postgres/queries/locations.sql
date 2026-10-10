-- name: CreateLocation :exec
INSERT INTO locations (id, tenant_id, name, active, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetLocation :one
SELECT id, tenant_id, name, active, created_at
FROM locations
WHERE id = $1;

-- name: ListLocations :many
SELECT id, tenant_id, name, active, created_at
FROM locations
ORDER BY name, id;
