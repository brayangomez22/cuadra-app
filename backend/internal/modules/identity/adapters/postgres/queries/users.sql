-- name: CreateUser :exec
INSERT INTO users (id, tenant_id, email, name, password_hash, role, active, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetUser :one
SELECT id, tenant_id, email, name, password_hash, role, active, created_at
FROM users
WHERE tenant_id = $1 AND id = $2;

-- name: GetUserByEmail :one
SELECT id, tenant_id, email, name, password_hash, role, active, created_at
FROM users
WHERE tenant_id = $1 AND email = $2;

-- name: UpdateUser :execrows
UPDATE users
SET name = $3, password_hash = $4, role = $5, active = $6
WHERE tenant_id = $1 AND id = $2;
