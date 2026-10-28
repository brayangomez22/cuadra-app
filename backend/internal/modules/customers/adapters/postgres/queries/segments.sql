-- name: CreateSegment :exec
INSERT INTO customer_segments (id, tenant_id, name, created_at)
VALUES (sqlc.arg(id), sqlc.arg(tenant_id), sqlc.arg(name), sqlc.arg(created_at));

-- name: RenameSegment :execrows
UPDATE customer_segments SET name = sqlc.arg(name) WHERE id = sqlc.arg(id);

-- name: GetSegment :one
SELECT id, tenant_id, name, created_at FROM customer_segments WHERE id = sqlc.arg(id);

-- name: ListSegments :many
SELECT id, tenant_id, name, created_at FROM customer_segments ORDER BY lower(name), id;

-- name: DeleteSegment :execrows
DELETE FROM customer_segments WHERE id = sqlc.arg(id);
