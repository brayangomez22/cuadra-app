-- name: InsertEvent :exec
INSERT INTO audit_events (id, tenant_id, user_id, role, action, entity_type, entity_id, location_id, occurred_at, details)
VALUES (sqlc.arg(id), sqlc.arg(tenant_id), sqlc.arg(user_id), sqlc.arg(role), sqlc.arg(action), sqlc.arg(entity_type),
        sqlc.narg(entity_id), sqlc.narg(location_id), sqlc.arg(occurred_at), sqlc.arg(details));

-- name: ListEvents :many
SELECT id, tenant_id, user_id, role, action, entity_id, location_id, occurred_at, details
FROM audit_events
WHERE (sqlc.narg(action)::text IS NULL OR action = sqlc.narg(action))
  AND (sqlc.narg(user_id)::uuid IS NULL OR user_id = sqlc.narg(user_id))
  AND (sqlc.narg(entity_id)::uuid IS NULL OR entity_id = sqlc.narg(entity_id))
  AND (sqlc.narg(occurred_from)::timestamptz IS NULL OR occurred_at >= sqlc.narg(occurred_from))
  AND (sqlc.narg(occurred_before)::timestamptz IS NULL OR occurred_at < sqlc.narg(occurred_before))
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountEvents :one
SELECT count(*)
FROM audit_events
WHERE (sqlc.narg(action)::text IS NULL OR action = sqlc.narg(action))
  AND (sqlc.narg(user_id)::uuid IS NULL OR user_id = sqlc.narg(user_id))
  AND (sqlc.narg(entity_id)::uuid IS NULL OR entity_id = sqlc.narg(entity_id))
  AND (sqlc.narg(occurred_from)::timestamptz IS NULL OR occurred_at >= sqlc.narg(occurred_from))
  AND (sqlc.narg(occurred_before)::timestamptz IS NULL OR occurred_at < sqlc.narg(occurred_before));
