-- name: UpsertReorderRule :exec
INSERT INTO reorder_rules (tenant_id, variant_id, location_id, min_quantity, max_quantity, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_id, location_id, variant_id)
DO UPDATE SET min_quantity = EXCLUDED.min_quantity, max_quantity = EXCLUDED.max_quantity, updated_at = EXCLUDED.updated_at;

-- name: DeleteReorderRule :execrows
DELETE FROM reorder_rules
WHERE variant_id = $1 AND location_id = $2;

-- name: ListReorderRulesForVariants :many
SELECT tenant_id, variant_id, location_id, min_quantity, max_quantity
FROM reorder_rules
WHERE location_id = sqlc.arg(location_id) AND variant_id = ANY(sqlc.arg(variant_ids)::uuid[]);

-- ListReorderAlerts pages the rules whose stock at their location (of every
-- lot, zero without a level) is at or below their minimum, the furthest below
-- it first. The condition is the one of domain.ReorderRule.NeedsReorder; the
-- integration tests check both agree.
-- name: ListReorderAlerts :many
SELECT r.tenant_id, r.variant_id, r.location_id, r.min_quantity, r.max_quantity,
       COALESCE(s.quantity, 0)::numeric(18,4) AS on_hand
FROM reorder_rules r
LEFT JOIN LATERAL (
    SELECT sum(l.quantity) AS quantity
    FROM stock_levels l
    WHERE l.variant_id = r.variant_id AND l.location_id = r.location_id
) s ON true
WHERE (sqlc.narg(location_id)::uuid IS NULL OR r.location_id = sqlc.narg(location_id))
  AND COALESCE(s.quantity, 0) <= r.min_quantity
ORDER BY COALESCE(s.quantity, 0) - r.min_quantity, r.location_id, r.variant_id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountReorderAlerts :one
SELECT count(*)
FROM reorder_rules r
LEFT JOIN LATERAL (
    SELECT sum(l.quantity) AS quantity
    FROM stock_levels l
    WHERE l.variant_id = r.variant_id AND l.location_id = r.location_id
) s ON true
WHERE (sqlc.narg(location_id)::uuid IS NULL OR r.location_id = sqlc.narg(location_id))
  AND COALESCE(s.quantity, 0) <= r.min_quantity;
