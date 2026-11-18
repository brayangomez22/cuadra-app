-- name: CreateInvitation :exec
INSERT INTO user_invitations (id, tenant_id, email, name, role, invited_by, token_hash, expires_at, created_at, accepted_at, revoked_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: GetInvitation :one
SELECT * FROM user_invitations WHERE tenant_id = $1 AND id = $2;

-- name: GetInvitationByHash :one
SELECT * FROM user_invitations WHERE tenant_id = $1 AND token_hash = $2;

-- name: ListOpenInvitations :many
SELECT * FROM user_invitations
WHERE tenant_id = $1 AND accepted_at IS NULL AND revoked_at IS NULL
ORDER BY created_at DESC, id DESC;

-- CloseInvitation stores the acceptance or revocation; it affects no row
-- if the invitation was already closed.
-- name: CloseInvitation :execrows
UPDATE user_invitations
SET accepted_at = sqlc.narg(accepted_at), revoked_at = sqlc.narg(revoked_at)
WHERE tenant_id = sqlc.arg(tenant_id) AND id = sqlc.arg(id) AND accepted_at IS NULL AND revoked_at IS NULL;

-- name: RevokeOpenInvitations :exec
UPDATE user_invitations
SET revoked_at = sqlc.arg(revoked_at)
WHERE tenant_id = sqlc.arg(tenant_id) AND email = sqlc.arg(email) AND accepted_at IS NULL AND revoked_at IS NULL;
