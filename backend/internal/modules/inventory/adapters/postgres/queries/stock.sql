-- EnsureStockLevel creates an empty level if there is none, so the next
-- statement can lock it. It fails on the foreign keys if the product or the
-- location does not exist in the tenant.
-- name: EnsureStockLevel :exec
INSERT INTO stock_levels (tenant_id, product_id, location_id, quantity, average_cost, updated_at)
VALUES ($1, $2, $3, 0, 0, $4)
ON CONFLICT (tenant_id, product_id, location_id) DO NOTHING;

-- LockStockLevels locks the levels of a product at the given locations, in
-- location order, until the transaction ends.
-- name: LockStockLevels :many
SELECT tenant_id, product_id, location_id, quantity, average_cost
FROM stock_levels
WHERE product_id = sqlc.arg(product_id) AND location_id = ANY(sqlc.arg(location_ids)::uuid[])
ORDER BY location_id
FOR UPDATE;

-- name: UpdateStockLevel :execrows
UPDATE stock_levels
SET quantity = $3, average_cost = $4, updated_at = $5
WHERE product_id = $1 AND location_id = $2;

-- name: InsertStockMovement :exec
INSERT INTO stock_movements (id, tenant_id, product_id, location_id, type, quantity, unit_cost, balance_after,
                             average_cost_after, reason, reference_kind, reference_id, user_id, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);

-- name: ListStockLevels :many
SELECT tenant_id, product_id, location_id, quantity, average_cost
FROM stock_levels
WHERE location_id = sqlc.arg(location_id)
  AND (sqlc.narg(product_id)::uuid IS NULL OR product_id = sqlc.narg(product_id))
ORDER BY product_id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountStockLevels :one
SELECT count(*)
FROM stock_levels
WHERE location_id = sqlc.arg(location_id)
  AND (sqlc.narg(product_id)::uuid IS NULL OR product_id = sqlc.narg(product_id));

-- ListKardex pages a product's movements, newest first.
-- name: ListKardex :many
SELECT id, tenant_id, product_id, location_id, type, quantity, unit_cost, balance_after,
       average_cost_after, reason, reference_kind, reference_id, user_id, occurred_at
FROM stock_movements
WHERE product_id = sqlc.arg(product_id)
  AND (sqlc.narg(location_id)::uuid IS NULL OR location_id = sqlc.narg(location_id))
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountKardex :one
SELECT count(*)
FROM stock_movements
WHERE product_id = sqlc.arg(product_id)
  AND (sqlc.narg(location_id)::uuid IS NULL OR location_id = sqlc.narg(location_id));
