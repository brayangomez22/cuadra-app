-- The stock reports: idle stock and stock adjustments.

-- IdleVariants returns the variants with stock (at the location, or adding
-- up every location) not sold there since since_start, or never: their
-- quantity in the base unit, its value at the average cost and their last
-- sale. Inactive products count too: their stock is idle money as well.
-- name: IdleVariants :many
SELECT p.id AS product_id, p.name AS product_name, u.code AS unit_code, v.id AS variant_id,
       axes.label::text AS label, v.sku,
       sum(sl.quantity)::numeric(18,4) AS quantity,
       sum(sl.quantity * sl.average_cost)::numeric(18,4) AS value,
       coalesce(max(CASE WHEN sqlc.narg(location_id)::uuid IS NULL THEN sold_any.at ELSE sold_here.at END), 'epoch')::timestamptz AS last_sold_at
FROM stock_levels sl
JOIN product_variants v ON v.tenant_id = sl.tenant_id AND v.id = sl.variant_id
JOIN products p ON p.tenant_id = v.tenant_id AND p.id = v.product_id
JOIN units_of_measure u ON u.tenant_id = p.tenant_id AND u.id = p.base_unit_id
CROSS JOIN LATERAL (SELECT coalesce(string_agg(a.value, ' · ' ORDER BY a.key COLLATE "C"), '') AS label
                    FROM jsonb_each_text(v.axis_values) AS a) axes
CROSS JOIN LATERAL (SELECT max(m.occurred_at) AS at FROM stock_movements m
                    WHERE m.tenant_id = sl.tenant_id AND m.variant_id = sl.variant_id AND m.type = 'sale_out') sold_any
CROSS JOIN LATERAL (SELECT max(m.occurred_at) AS at FROM stock_movements m
                    WHERE m.tenant_id = sl.tenant_id AND m.variant_id = sl.variant_id AND m.type = 'sale_out'
                      AND m.location_id = sl.location_id) sold_here
WHERE sl.quantity > 0
  AND (sqlc.narg(location_id)::uuid IS NULL OR sl.location_id = sqlc.narg(location_id))
  AND (sqlc.narg(category_id)::uuid IS NULL OR p.category_id = sqlc.narg(category_id))
  AND (sqlc.narg(attribute)::text IS NULL OR p.attributes ->> sqlc.narg(attribute)::text = sqlc.narg(attribute_value)::text)
  AND coalesce(CASE WHEN sqlc.narg(location_id)::uuid IS NULL THEN sold_any.at ELSE sold_here.at END < sqlc.arg(since_start)::timestamptz, true)
GROUP BY p.id, p.name, u.code, v.id, axes.label, v.sku
ORDER BY p.name, p.id, v.id;

-- AdjustmentsByUser adds up the period's adjustments (manual and from
-- stock counts) by who made them. Value is at the cost of the movement:
-- negative when stock was lost.
-- name: AdjustmentsByUser :many
SELECT m.user_id::text AS key, coalesce(max(u.name), '')::text AS label,
       count(*)::bigint AS adjustments,
       coalesce(sum(m.quantity) FILTER (WHERE m.quantity > 0), 0)::numeric(18,4) AS quantity_in,
       coalesce(sum(-m.quantity) FILTER (WHERE m.quantity < 0), 0)::numeric(18,4) AS quantity_out,
       coalesce(sum(m.quantity * m.unit_cost), 0)::numeric(18,4) AS value
FROM stock_movements m
LEFT JOIN users u ON u.tenant_id = m.tenant_id AND u.id = m.user_id
WHERE m.type = 'adjustment'
  AND m.occurred_at >= sqlc.arg(period_start) AND m.occurred_at < sqlc.arg(period_end)
  AND (sqlc.narg(location_id)::uuid IS NULL OR m.location_id = sqlc.narg(location_id))
GROUP BY m.user_id
ORDER BY value, label, key;

-- AdjustmentsByVariant adds up the period's adjustments by variant, named
-- by its product and the values of its axes, in the base unit.
-- name: AdjustmentsByVariant :many
SELECT m.variant_id::text AS key, max(p.name)::text AS product_name, max(axes.label)::text AS label,
       max(u.code)::text AS unit_code,
       count(*)::bigint AS adjustments,
       coalesce(sum(m.quantity) FILTER (WHERE m.quantity > 0), 0)::numeric(18,4) AS quantity_in,
       coalesce(sum(-m.quantity) FILTER (WHERE m.quantity < 0), 0)::numeric(18,4) AS quantity_out,
       coalesce(sum(m.quantity * m.unit_cost), 0)::numeric(18,4) AS value
FROM stock_movements m
JOIN product_variants v ON v.tenant_id = m.tenant_id AND v.id = m.variant_id
JOIN products p ON p.tenant_id = v.tenant_id AND p.id = v.product_id
JOIN units_of_measure u ON u.tenant_id = p.tenant_id AND u.id = p.base_unit_id
CROSS JOIN LATERAL (SELECT coalesce(string_agg(a.value, ' · ' ORDER BY a.key COLLATE "C"), '') AS label
                    FROM jsonb_each_text(v.axis_values) AS a) axes
WHERE m.type = 'adjustment'
  AND m.occurred_at >= sqlc.arg(period_start) AND m.occurred_at < sqlc.arg(period_end)
  AND (sqlc.narg(location_id)::uuid IS NULL OR m.location_id = sqlc.narg(location_id))
GROUP BY m.variant_id
ORDER BY value, product_name, key;
