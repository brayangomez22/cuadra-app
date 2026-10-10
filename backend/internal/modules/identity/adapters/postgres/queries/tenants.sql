-- name: CreateTenant :exec
INSERT INTO tenants (id, name, nit, status, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetTenant :one
SELECT id, name, nit, status, created_at
FROM tenants
WHERE id = $1;

-- name: UpdateTenant :execrows
UPDATE tenants
SET name = $2, status = $3
WHERE id = $1;
