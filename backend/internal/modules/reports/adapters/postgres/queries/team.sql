-- What the team did: cash closings and the control of each user.

-- CashClosings returns each closing of the period (by the day it closed)
-- with its cashier, location and what it sold (as the closing report: it
-- does not change if a sale is voided afterwards). The adapter groups them.
-- name: CashClosings :many
SELECT cs.id, cs.user_id, coalesce(u.name, '')::text AS user_name, cs.location_id, l.name AS location_name,
       to_char(cs.closed_at AT TIME ZONE 'America/Bogota', 'YYYY-MM-DD')::text AS closed_day,
       coalesce((SELECT sum(p.amount) FROM cash_session_payments p
                 WHERE p.tenant_id = cs.tenant_id AND p.session_id = cs.id), 0)::numeric(18,4) AS sales_total,
       coalesce(cs.difference, 0)::numeric(18,4) AS difference
FROM cash_sessions cs
JOIN locations l ON l.tenant_id = cs.tenant_id AND l.id = cs.location_id
LEFT JOIN users u ON u.tenant_id = cs.tenant_id AND u.id = cs.user_id
WHERE cs.status = 'closed'
  AND cs.closed_at >= sqlc.arg(period_start)::timestamptz AND cs.closed_at < sqlc.arg(period_end)::timestamptz
  AND (sqlc.narg(location_id)::uuid IS NULL OR cs.location_id = sqlc.narg(location_id))
ORDER BY cs.closed_at, cs.id;

-- TeamControl adds up, by user and bucket (day, week from Monday or month,
-- in Colombia), the voids, discounts and price changes of the audit log,
-- the stock adjustments of the kardex (at the cost of the movement) and the
-- cash closings with a difference (by their cashier).
-- name: TeamControl :many
WITH events AS (
    SELECT e.user_id, date_trunc(sqlc.arg(bucket)::text, e.occurred_at AT TIME ZONE 'America/Bogota')::date AS bucket_start,
           count(*) FILTER (WHERE e.action = 'sale.voided') AS voids,
           coalesce(sum((e.details ->> 'total')::numeric) FILTER (WHERE e.action = 'sale.voided'), 0) AS voids_total,
           count(*) FILTER (WHERE e.action = 'sale.discount_applied') AS discounts,
           coalesce(sum((e.details ->> 'discount')::numeric) FILTER (WHERE e.action = 'sale.discount_applied'), 0) AS discounts_amount,
           coalesce(sum((e.details ->> 'gross')::numeric) FILTER (WHERE e.action = 'sale.discount_applied'), 0) AS discounts_gross,
           count(*) FILTER (WHERE e.action = 'variant.price_changed') AS price_changes
    FROM audit_events e
    WHERE e.action IN ('sale.voided', 'sale.discount_applied', 'variant.price_changed')
      AND e.occurred_at >= sqlc.arg(period_start) AND e.occurred_at < sqlc.arg(period_end)
      AND (sqlc.narg(location_id)::uuid IS NULL OR e.location_id = sqlc.narg(location_id))
    GROUP BY 1, 2
), adjustments AS (
    SELECT m.user_id, date_trunc(sqlc.arg(bucket)::text, m.occurred_at AT TIME ZONE 'America/Bogota')::date AS bucket_start,
           count(*) AS adjustments, coalesce(sum(m.quantity * m.unit_cost), 0) AS adjustments_value
    FROM stock_movements m
    WHERE m.type = 'adjustment'
      AND m.occurred_at >= sqlc.arg(period_start) AND m.occurred_at < sqlc.arg(period_end)
      AND (sqlc.narg(location_id)::uuid IS NULL OR m.location_id = sqlc.narg(location_id))
    GROUP BY 1, 2
), cash AS (
    SELECT cs.user_id, date_trunc(sqlc.arg(bucket)::text, cs.closed_at AT TIME ZONE 'America/Bogota')::date AS bucket_start,
           count(*) AS cash_differences,
           coalesce(sum(-cs.difference) FILTER (WHERE cs.difference < 0), 0) AS cash_shortage,
           coalesce(sum(cs.difference) FILTER (WHERE cs.difference > 0), 0) AS cash_overage
    FROM cash_sessions cs
    WHERE cs.status = 'closed' AND cs.difference <> 0
      AND cs.closed_at >= sqlc.arg(period_start) AND cs.closed_at < sqlc.arg(period_end)
      AND (sqlc.narg(location_id)::uuid IS NULL OR cs.location_id = sqlc.narg(location_id))
    GROUP BY 1, 2
), keys AS (
    SELECT user_id, bucket_start FROM events
    UNION SELECT user_id, bucket_start FROM adjustments
    UNION SELECT user_id, bucket_start FROM cash
)
SELECT k.user_id, coalesce(u.name, '')::text AS name, k.bucket_start::date AS bucket_start,
       coalesce(ev.voids, 0)::bigint AS voids, coalesce(ev.voids_total, 0)::numeric(18,4) AS voids_total,
       coalesce(ev.discounts, 0)::bigint AS discounts,
       coalesce(ev.discounts_amount, 0)::numeric(18,4) AS discounts_amount,
       coalesce(ev.discounts_gross, 0)::numeric(18,4) AS discounts_gross,
       coalesce(ev.price_changes, 0)::bigint AS price_changes,
       coalesce(ad.adjustments, 0)::bigint AS adjustments,
       coalesce(ad.adjustments_value, 0)::numeric(18,4) AS adjustments_value,
       coalesce(ca.cash_differences, 0)::bigint AS cash_differences,
       coalesce(ca.cash_shortage, 0)::numeric(18,4) AS cash_shortage,
       coalesce(ca.cash_overage, 0)::numeric(18,4) AS cash_overage
FROM keys k
LEFT JOIN events ev ON ev.user_id = k.user_id AND ev.bucket_start = k.bucket_start
LEFT JOIN adjustments ad ON ad.user_id = k.user_id AND ad.bucket_start = k.bucket_start
LEFT JOIN cash ca ON ca.user_id = k.user_id AND ca.bucket_start = k.bucket_start
LEFT JOIN users u ON u.id = k.user_id
ORDER BY k.bucket_start, name, k.user_id;
