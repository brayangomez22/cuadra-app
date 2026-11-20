-- name: GetDismissal :one
SELECT dismissed_at, pending_steps
FROM onboarding_dismissals
WHERE tenant_id = sqlc.arg(tenant_id) AND user_id = sqlc.arg(user_id);

-- name: SaveDismissal :exec
INSERT INTO onboarding_dismissals (tenant_id, user_id, dismissed_at, pending_steps)
VALUES (sqlc.arg(tenant_id), sqlc.arg(user_id), sqlc.arg(dismissed_at), sqlc.arg(pending_steps)::text[])
ON CONFLICT (tenant_id, user_id)
DO UPDATE SET dismissed_at = EXCLUDED.dismissed_at, pending_steps = EXCLUDED.pending_steps;
