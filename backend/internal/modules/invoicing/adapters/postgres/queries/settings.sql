-- name: GetInvoicingSettings :one
SELECT enabled, default_kind FROM invoicing_settings WHERE tenant_id = $1;

-- name: SaveInvoicingSettings :exec
INSERT INTO invoicing_settings (tenant_id, enabled, default_kind, updated_at)
VALUES ($1, $2, $3, statement_timestamp())
ON CONFLICT (tenant_id) DO UPDATE
SET enabled = EXCLUDED.enabled, default_kind = EXCLUDED.default_kind, updated_at = EXCLUDED.updated_at;
