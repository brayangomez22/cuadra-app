-- name: GetStockPolicy :one
SELECT allow_negative_stock
FROM inventory_settings
WHERE tenant_id = $1;

-- name: SaveStockPolicy :exec
INSERT INTO inventory_settings (tenant_id, allow_negative_stock)
VALUES ($1, $2)
ON CONFLICT (tenant_id) DO UPDATE SET allow_negative_stock = EXCLUDED.allow_negative_stock;
