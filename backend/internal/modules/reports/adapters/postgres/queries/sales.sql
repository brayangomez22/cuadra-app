-- The sales reports. Every query reads the confirmed sales of a period
-- ([period_start, period_end), the days in Colombia) and, optionally, of a
-- location; a voided sale counts nowhere. Net is without VAT and after
-- discounts; total = net + tax, as each sale stores them.

-- name: SalesTotals :one
SELECT count(*)::bigint AS sales_count,
       coalesce(sum(s.subtotal - s.discounts), 0)::numeric(18,4) AS net,
       coalesce(sum(s.tax), 0)::numeric(18,4) AS tax,
       coalesce(sum(s.total), 0)::numeric(18,4) AS total
FROM sales s
WHERE s.status = 'confirmed'
  AND s.confirmed_at >= sqlc.arg(period_start) AND s.confirmed_at < sqlc.arg(period_end)
  AND (sqlc.narg(location_id)::uuid IS NULL OR s.location_id = sqlc.narg(location_id));

-- name: SalesByDay :many
SELECT to_char(s.confirmed_at AT TIME ZONE 'America/Bogota', 'YYYY-MM-DD')::text AS key,
       count(*)::bigint AS sales_count,
       coalesce(sum(s.subtotal - s.discounts), 0)::numeric(18,4) AS net,
       coalesce(sum(s.tax), 0)::numeric(18,4) AS tax,
       coalesce(sum(s.total), 0)::numeric(18,4) AS total
FROM sales s
WHERE s.status = 'confirmed'
  AND s.confirmed_at >= sqlc.arg(period_start) AND s.confirmed_at < sqlc.arg(period_end)
  AND (sqlc.narg(location_id)::uuid IS NULL OR s.location_id = sqlc.narg(location_id))
GROUP BY 1
ORDER BY 1;

-- name: SalesByUser :many
SELECT s.user_id::text AS key, coalesce(max(u.name), '')::text AS label,
       count(*)::bigint AS sales_count,
       coalesce(sum(s.subtotal - s.discounts), 0)::numeric(18,4) AS net,
       coalesce(sum(s.tax), 0)::numeric(18,4) AS tax,
       coalesce(sum(s.total), 0)::numeric(18,4) AS total
FROM sales s
LEFT JOIN users u ON u.tenant_id = s.tenant_id AND u.id = s.user_id
WHERE s.status = 'confirmed'
  AND s.confirmed_at >= sqlc.arg(period_start) AND s.confirmed_at < sqlc.arg(period_end)
  AND (sqlc.narg(location_id)::uuid IS NULL OR s.location_id = sqlc.narg(location_id))
GROUP BY s.user_id
ORDER BY total DESC, label, key;

-- name: SalesByLocation :many
SELECT s.location_id::text AS key, coalesce(max(l.name), '')::text AS label,
       count(*)::bigint AS sales_count,
       coalesce(sum(s.subtotal - s.discounts), 0)::numeric(18,4) AS net,
       coalesce(sum(s.tax), 0)::numeric(18,4) AS tax,
       coalesce(sum(s.total), 0)::numeric(18,4) AS total
FROM sales s
JOIN locations l ON l.tenant_id = s.tenant_id AND l.id = s.location_id
WHERE s.status = 'confirmed'
  AND s.confirmed_at >= sqlc.arg(period_start) AND s.confirmed_at < sqlc.arg(period_end)
  AND (sqlc.narg(location_id)::uuid IS NULL OR s.location_id = sqlc.narg(location_id))
GROUP BY s.location_id
ORDER BY total DESC, label, key;

-- SalesByCategory adds up the lines by their product's current category
-- (an empty key without one).
-- name: SalesByCategory :many
SELECT coalesce(c.id::text, '')::text AS key, coalesce(max(c.name), '')::text AS label,
       count(DISTINCT s.id)::bigint AS sales_count,
       coalesce(sum(sl.gross - sl.discount), 0)::numeric(18,4) AS net,
       coalesce(sum(sl.tax), 0)::numeric(18,4) AS tax,
       coalesce(sum(sl.gross - sl.discount + sl.tax), 0)::numeric(18,4) AS total
FROM sales s
JOIN sale_lines sl ON sl.tenant_id = s.tenant_id AND sl.sale_id = s.id
JOIN products p ON p.tenant_id = sl.tenant_id AND p.id = sl.product_id
LEFT JOIN categories c ON c.tenant_id = p.tenant_id AND c.id = p.category_id
WHERE s.status = 'confirmed'
  AND s.confirmed_at >= sqlc.arg(period_start) AND s.confirmed_at < sqlc.arg(period_end)
  AND (sqlc.narg(location_id)::uuid IS NULL OR s.location_id = sqlc.narg(location_id))
GROUP BY c.id
ORDER BY total DESC, label, key;

-- SalesByPaymentMethod adds up what each method paid: cash without the
-- change, the only method that gives it.
-- name: SalesByPaymentMethod :many
SELECT sp.method::text AS key,
       count(DISTINCT s.id)::bigint AS sales_count,
       coalesce(sum(sp.amount - CASE WHEN sp.method = 'cash' THEN s.change ELSE 0 END), 0)::numeric(18,4) AS total
FROM sales s
JOIN sale_payments sp ON sp.tenant_id = s.tenant_id AND sp.sale_id = s.id
WHERE s.status = 'confirmed'
  AND s.confirmed_at >= sqlc.arg(period_start) AND s.confirmed_at < sqlc.arg(period_end)
  AND (sqlc.narg(location_id)::uuid IS NULL OR s.location_id = sqlc.narg(location_id))
GROUP BY sp.method
ORDER BY total DESC, key;

-- SoldVariants adds up each variant's sales of the period: the quantity in
-- its product's base unit, the revenue without VAT and the cost of its
-- sale_out movements, at the average cost when it was sold. The label is
-- the one frozen in the sale (empty for a product without variants).
-- name: SoldVariants :many
WITH sold AS (
    SELECT s.tenant_id, s.id AS sale_id, sl.variant_id,
           sum(sl.quantity * sl.unit_factor) AS quantity,
           sum(sl.gross - sl.discount) AS revenue,
           max(sl.variant_label) AS label
    FROM sales s
    JOIN sale_lines sl ON sl.tenant_id = s.tenant_id AND sl.sale_id = s.id
    WHERE s.status = 'confirmed'
      AND s.confirmed_at >= sqlc.arg(period_start) AND s.confirmed_at < sqlc.arg(period_end)
      AND (sqlc.narg(location_id)::uuid IS NULL OR s.location_id = sqlc.narg(location_id))
    GROUP BY s.tenant_id, s.id, sl.variant_id
), by_variant AS (
    SELECT sold.variant_id, sum(sold.quantity) AS quantity, sum(sold.revenue) AS revenue,
           coalesce(max(sold.label), '') AS label,
           sum(coalesce((SELECT sum(-m.quantity * m.unit_cost) FROM stock_movements m
                         WHERE m.tenant_id = sold.tenant_id AND m.reference_kind = 'sale' AND m.reference_id = sold.sale_id
                           AND m.variant_id = sold.variant_id AND m.type = 'sale_out'), 0)) AS cost
    FROM sold
    GROUP BY sold.variant_id
)
SELECT p.id AS product_id, p.name AS product_name, u.code AS unit_code, v.id AS variant_id,
       b.label::text AS label, v.sku,
       b.quantity::numeric(18,4) AS quantity, b.revenue::numeric(18,4) AS revenue, b.cost::numeric(18,4) AS cost
FROM by_variant b
JOIN product_variants v ON v.id = b.variant_id
JOIN products p ON p.tenant_id = v.tenant_id AND p.id = v.product_id
JOIN units_of_measure u ON u.tenant_id = p.tenant_id AND u.id = p.base_unit_id
WHERE (sqlc.narg(category_id)::uuid IS NULL OR p.category_id = sqlc.narg(category_id))
  AND (sqlc.narg(attribute)::text IS NULL OR p.attributes ->> sqlc.narg(attribute)::text = sqlc.narg(attribute_value)::text)
ORDER BY p.name, p.id, v.id;

-- SaleProfitsPage is a page of the period's sales with their revenue
-- without VAT and the cost of their sale_out movements, newest first, with
-- the count of all of them.
-- name: SaleProfitsPage :many
SELECT s.id AS sale_id, s.number, s.location_id, l.name AS location_name, s.user_id,
       coalesce(u.name, '')::text AS user_name, s.confirmed_at,
       (s.subtotal - s.discounts)::numeric(18,4) AS revenue,
       coalesce((SELECT sum(-m.quantity * m.unit_cost) FROM stock_movements m
                 WHERE m.tenant_id = s.tenant_id AND m.reference_kind = 'sale' AND m.reference_id = s.id AND m.type = 'sale_out'), 0)::numeric(18,4) AS cost,
       count(*) OVER ()::bigint AS total
FROM sales s
JOIN locations l ON l.tenant_id = s.tenant_id AND l.id = s.location_id
LEFT JOIN users u ON u.tenant_id = s.tenant_id AND u.id = s.user_id
WHERE s.status = 'confirmed'
  AND s.confirmed_at >= sqlc.arg(period_start) AND s.confirmed_at < sqlc.arg(period_end)
  AND (sqlc.narg(location_id)::uuid IS NULL OR s.location_id = sqlc.narg(location_id))
ORDER BY s.confirmed_at DESC, s.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: SaleProfitsTotals :one
SELECT count(*)::bigint AS sales_count,
       coalesce(sum(s.subtotal - s.discounts), 0)::numeric(18,4) AS revenue,
       coalesce(sum((SELECT sum(-m.quantity * m.unit_cost) FROM stock_movements m
                     WHERE m.tenant_id = s.tenant_id AND m.reference_kind = 'sale' AND m.reference_id = s.id AND m.type = 'sale_out')), 0)::numeric(18,4) AS cost
FROM sales s
WHERE s.status = 'confirmed'
  AND s.confirmed_at >= sqlc.arg(period_start) AND s.confirmed_at < sqlc.arg(period_end)
  AND (sqlc.narg(location_id)::uuid IS NULL OR s.location_id = sqlc.narg(location_id));
