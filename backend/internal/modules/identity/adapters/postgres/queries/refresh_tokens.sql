-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (id, tenant_id, user_id, family_id, token_hash, expires_at, created_at, revoked_at, family_started_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetRefreshTokenByHash :one
SELECT id, tenant_id, user_id, family_id, token_hash, expires_at, created_at, revoked_at, family_started_at
FROM refresh_tokens
WHERE tenant_id = $1 AND token_hash = $2;

-- RevokeRefreshToken revokes a token only if it is still live: 0 rows means
-- another transaction revoked it first.
-- name: RevokeRefreshToken :execrows
UPDATE refresh_tokens
SET revoked_at = $3
WHERE tenant_id = $1 AND id = $2 AND revoked_at IS NULL;

-- name: RevokeRefreshTokenFamily :execrows
UPDATE refresh_tokens
SET revoked_at = $3
WHERE tenant_id = $1 AND family_id = $2 AND revoked_at IS NULL;

-- TenantsWithExpiredRefreshTokens lists, across tenants, those with tokens
-- expired before $1 (00035, SECURITY DEFINER): only their ids.
-- name: TenantsWithExpiredRefreshTokens :many
SELECT tenant_id::uuid FROM tenants_with_expired_refresh_tokens(sqlc.arg(before)::timestamptz, sqlc.arg(max_tenants)::int);

-- name: DeleteExpiredRefreshTokens :execrows
DELETE FROM refresh_tokens
WHERE expires_at < sqlc.arg(before);

-- name: RevokeUserRefreshTokens :execrows
UPDATE refresh_tokens
SET revoked_at = sqlc.arg(revoked_at)
WHERE tenant_id = sqlc.arg(tenant_id) AND user_id = sqlc.arg(user_id) AND revoked_at IS NULL;
