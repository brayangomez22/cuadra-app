-- name: GetStockPolicy :one
SELECT allow_negative_stock
FROM inventory_settings
WHERE tenant_id = $1;
