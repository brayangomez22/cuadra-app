-- name: GetSalesPolicy :one
SELECT max_discount
FROM sales_settings
WHERE tenant_id = $1;
