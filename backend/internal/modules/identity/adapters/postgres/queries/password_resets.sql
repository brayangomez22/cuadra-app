-- SpendUnusedPasswordResets spends a user's unused resets before a new
-- one is issued: only the last link works.
-- name: SpendUnusedPasswordResets :exec
UPDATE password_resets
SET used_at = sqlc.arg(used_at)
WHERE tenant_id = sqlc.arg(tenant_id) AND user_id = sqlc.arg(user_id) AND used_at IS NULL;

-- name: CreatePasswordReset :exec
INSERT INTO password_resets (id, tenant_id, user_id, issued_by, token_hash, expires_at, created_at, used_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetPasswordResetByHash :one
SELECT * FROM password_resets WHERE tenant_id = $1 AND token_hash = $2;

-- name: MarkPasswordResetUsed :execrows
UPDATE password_resets
SET used_at = sqlc.arg(used_at)
WHERE tenant_id = sqlc.arg(tenant_id) AND id = sqlc.arg(id) AND used_at IS NULL;
