-- name: GetProviderAccount :one
SELECT mode, test_set_id, token_ciphertext FROM invoicing_provider_accounts WHERE tenant_id = $1;

-- name: SaveProviderAccount :exec
INSERT INTO invoicing_provider_accounts (tenant_id, mode, test_set_id, token_ciphertext, updated_at)
VALUES ($1, $2, $3, $4, statement_timestamp())
ON CONFLICT (tenant_id) DO UPDATE
SET mode = EXCLUDED.mode, test_set_id = EXCLUDED.test_set_id, token_ciphertext = EXCLUDED.token_ciphertext,
    updated_at = EXCLUDED.updated_at;
