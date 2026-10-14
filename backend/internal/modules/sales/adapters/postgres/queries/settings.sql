-- name: GetSalesPolicy :one
SELECT max_discount, require_cash_session
FROM sales_settings
WHERE tenant_id = $1;
