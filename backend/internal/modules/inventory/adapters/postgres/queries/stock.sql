-- The core works on the stock without a lot (lot_id NULL); the lots_expiry
-- module will add the queries of the stock with a lot. The movement queries
-- select lot_id only to read the whole row (sqlcgen.StockMovement).

-- EnsureStockLevel creates an empty level without a lot if there is none, so
-- the next statement can lock it. Two concurrent calls create one level: the
-- conflict is on stock_levels_key, whose NULLS NOT DISTINCT makes NULL lots
-- equal. It fails on the foreign keys if the variant or the location does not
-- exist in the tenant.
-- name: EnsureStockLevel :exec
INSERT INTO stock_levels (tenant_id, variant_id, location_id, lot_id, quantity, average_cost, updated_at)
VALUES ($1, $2, $3, NULL, 0, 0, $4)
ON CONFLICT ON CONSTRAINT stock_levels_key DO NOTHING;

-- LockStockLevels locks the levels without a lot of a variant at the given
-- locations, in location order, until the transaction ends.
-- name: LockStockLevels :many
SELECT tenant_id, variant_id, location_id, quantity, average_cost
FROM stock_levels
WHERE variant_id = sqlc.arg(variant_id) AND location_id = ANY(sqlc.arg(location_ids)::uuid[]) AND lot_id IS NULL
ORDER BY location_id
FOR UPDATE;

-- name: UpdateStockLevel :execrows
UPDATE stock_levels
SET quantity = $3, average_cost = $4, updated_at = $5
WHERE variant_id = $1 AND location_id = $2 AND lot_id IS NULL;

-- name: InsertStockMovement :exec
INSERT INTO stock_movements (id, tenant_id, variant_id, location_id, type, quantity, unit_cost, balance_after,
                             average_cost_after, reason, reference_kind, reference_id, user_id, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);

-- name: ListStockLevels :many
SELECT tenant_id, variant_id, location_id, quantity, average_cost
FROM stock_levels
WHERE location_id = sqlc.arg(location_id) AND lot_id IS NULL
  AND (sqlc.narg(variant_id)::uuid IS NULL OR variant_id = sqlc.narg(variant_id))
ORDER BY variant_id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountStockLevels :one
SELECT count(*)
FROM stock_levels
WHERE location_id = sqlc.arg(location_id) AND lot_id IS NULL
  AND (sqlc.narg(variant_id)::uuid IS NULL OR variant_id = sqlc.narg(variant_id));

-- ListKardex pages a variant's movements, newest first.
-- name: ListKardex :many
SELECT id, tenant_id, variant_id, location_id, type, quantity, unit_cost, balance_after,
       average_cost_after, reason, reference_kind, reference_id, user_id, occurred_at, lot_id
FROM stock_movements
WHERE variant_id = sqlc.arg(variant_id)
  AND (sqlc.narg(location_id)::uuid IS NULL OR location_id = sqlc.narg(location_id))
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountKardex :one
SELECT count(*)
FROM stock_movements
WHERE variant_id = sqlc.arg(variant_id)
  AND (sqlc.narg(location_id)::uuid IS NULL OR location_id = sqlc.narg(location_id));

-- ListMovementsByReference returns the movements of a document (a sale's
-- exits and the stock given back when it was voided), oldest first.
-- name: ListMovementsByReference :many
SELECT id, tenant_id, variant_id, location_id, type, quantity, unit_cost, balance_after,
       average_cost_after, reason, reference_kind, reference_id, user_id, occurred_at, lot_id
FROM stock_movements
WHERE reference_kind = sqlc.arg(reference_kind)::text AND reference_id = sqlc.arg(reference_id)::uuid
ORDER BY occurred_at, id;

-- VariantsHaveMovements reports whether any of the given variants has a
-- stock movement.
-- name: VariantsHaveMovements :one
SELECT EXISTS (SELECT 1 FROM stock_movements WHERE variant_id = ANY(sqlc.arg(variant_ids)::uuid[]));
