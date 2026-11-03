-- name: GetSalesPolicy :one
SELECT max_discount, require_cash_session
FROM sales_settings
WHERE tenant_id = $1;

-- name: SaveSalesPolicy :exec
INSERT INTO sales_settings (tenant_id, max_discount, require_cash_session)
VALUES ($1, $2, $3)
ON CONFLICT (tenant_id) DO UPDATE
SET max_discount = EXCLUDED.max_discount, require_cash_session = EXCLUDED.require_cash_session;
