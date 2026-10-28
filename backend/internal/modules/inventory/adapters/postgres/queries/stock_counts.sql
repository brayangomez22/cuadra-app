-- name: InsertStockCount :exec
INSERT INTO stock_counts (id, tenant_id, location_id, category_ids, mode, status, started_by, started_at)
VALUES ($1, $2, $3, $4, $5, 'open', $6, $7);

-- SnapshotLevels takes a line per variant with a level at the location: the
-- snapshot of a full count. The start of a line is the variant's stock at the
-- location (without a lot) when the count started, according to the kardex:
-- the sum of its movements up to then. The sheet adds the movements after it
-- on the same clock, so what is expected never counts a movement twice.
-- name: SnapshotLevels :execrows
INSERT INTO stock_count_lines (tenant_id, count_id, variant_id, start_quantity, start_cost)
SELECT l.tenant_id, sqlc.arg(count_id)::uuid, l.variant_id,
       COALESCE((SELECT sum(m.quantity) FROM stock_movements m
                 WHERE m.variant_id = l.variant_id AND m.location_id = l.location_id
                   AND m.lot_id IS NULL AND m.occurred_at <= sqlc.arg(started_at)), 0),
       l.average_cost
FROM stock_levels l
WHERE l.location_id = sqlc.arg(location_id) AND l.lot_id IS NULL;

-- SnapshotVariants takes a line per variant of variant_ids, with or without
-- a level: the snapshot of a partial count.
-- name: SnapshotVariants :execrows
INSERT INTO stock_count_lines (tenant_id, count_id, variant_id, start_quantity, start_cost)
SELECT sqlc.arg(tenant_id)::uuid, sqlc.arg(count_id)::uuid, v.variant_id,
       COALESCE((SELECT sum(m.quantity) FROM stock_movements m
                 WHERE m.variant_id = v.variant_id AND m.location_id = sqlc.arg(location_id)
                   AND m.lot_id IS NULL AND m.occurred_at <= sqlc.arg(started_at)), 0),
       COALESCE(l.average_cost, 0)
FROM (SELECT DISTINCT unnest(sqlc.arg(variant_ids)::uuid[]) AS variant_id) v
LEFT JOIN stock_levels l
       ON l.variant_id = v.variant_id AND l.location_id = sqlc.arg(location_id) AND l.lot_id IS NULL;

-- name: GetStockCount :one
SELECT id, tenant_id, location_id, category_ids, mode, status, started_by, started_at, closed_by, closed_at,
       COALESCE(lines_total, 0)::integer AS lines_total, COALESCE(lines_counted, 0)::integer AS lines_counted,
       COALESCE(lines_exact, 0)::integer AS lines_exact,
       COALESCE(net_difference_value, 0)::numeric(18,4) AS net_difference_value,
       COALESCE(absolute_difference_value, 0)::numeric(18,4) AS absolute_difference_value
FROM stock_counts
WHERE id = $1;

-- LockStockCount takes the count against new entries (they share it) until
-- the transaction ends.
-- name: LockStockCount :one
SELECT id FROM stock_counts WHERE id = $1 FOR UPDATE;

-- ShareStockCount reads the status of the count and keeps it from closing
-- until the transaction ends.
-- name: ShareStockCount :one
SELECT status FROM stock_counts WHERE id = $1 FOR SHARE;

-- name: ListStockCounts :many
SELECT id, tenant_id, location_id, category_ids, mode, status, started_by, started_at, closed_by, closed_at,
       COALESCE(lines_total, 0)::integer AS lines_total, COALESCE(lines_counted, 0)::integer AS lines_counted,
       COALESCE(lines_exact, 0)::integer AS lines_exact,
       COALESCE(net_difference_value, 0)::numeric(18,4) AS net_difference_value,
       COALESCE(absolute_difference_value, 0)::numeric(18,4) AS absolute_difference_value
FROM stock_counts
WHERE (sqlc.narg(location_id)::uuid IS NULL OR location_id = sqlc.narg(location_id))
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
ORDER BY started_at DESC, id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountStockCounts :one
SELECT count(*)
FROM stock_counts
WHERE (sqlc.narg(location_id)::uuid IS NULL OR location_id = sqlc.narg(location_id))
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status));

-- name: StockCountProgress :one
SELECT count(*)::integer AS lines,
       (count(*) FILTER (WHERE EXISTS (
           SELECT 1 FROM stock_count_entries e
           WHERE e.count_id = l.count_id AND e.variant_id = l.variant_id AND e.round = l.round)))::integer AS counted
FROM stock_count_lines l
WHERE l.count_id = $1;

-- StockCountSheet returns every line with the time of its last entry in its
-- round and the net of the kardex at the location after the count started
-- and up to that entry (zero while uncounted).
-- name: StockCountSheet :many
SELECT l.variant_id, l.start_quantity, l.start_cost, l.round,
       COALESCE((SELECT sum(m.quantity) FROM stock_movements m
                 WHERE m.variant_id = l.variant_id AND m.location_id = sqlc.arg(location_id)
                   AND m.lot_id IS NULL AND m.occurred_at > sqlc.arg(started_at)
                   AND m.occurred_at <= last.counted_at), 0)::numeric(18,4) AS movements
FROM stock_count_lines l
LEFT JOIN LATERAL (
    SELECT max(e.counted_at) AS counted_at
    FROM stock_count_entries e
    WHERE e.count_id = l.count_id AND e.variant_id = l.variant_id AND e.round = l.round
) last ON true
WHERE l.count_id = sqlc.arg(count_id)
ORDER BY l.variant_id;

-- StockCountTotals returns what each person counted of each variant in its
-- line's current round.
-- name: StockCountTotals :many
SELECT e.variant_id, e.counted_by, sum(e.quantity)::numeric(18,4) AS quantity
FROM stock_count_entries e
JOIN stock_count_lines l
  ON l.count_id = e.count_id AND l.variant_id = e.variant_id AND l.round = e.round
WHERE e.count_id = $1
GROUP BY e.variant_id, e.counted_by
ORDER BY e.variant_id, e.counted_by;

-- name: RecountLines :execrows
UPDATE stock_count_lines
SET round = round + 1
WHERE count_id = sqlc.arg(count_id) AND variant_id = ANY(sqlc.arg(variant_ids)::uuid[]);

-- name: ApproveStockCount :exec
UPDATE stock_counts
SET status = 'approved', closed_by = sqlc.arg(closed_by)::uuid, closed_at = sqlc.arg(closed_at)::timestamptz,
    lines_total = sqlc.arg(lines_total)::bigint, lines_counted = sqlc.arg(lines_counted)::bigint,
    lines_exact = sqlc.arg(lines_exact)::bigint,
    net_difference_value = sqlc.arg(net_difference_value)::numeric(18,4),
    absolute_difference_value = sqlc.arg(absolute_difference_value)::numeric(18,4)
WHERE id = sqlc.arg(id);

-- name: CancelStockCount :exec
UPDATE stock_counts
SET status = 'canceled', closed_by = sqlc.arg(closed_by)::uuid, closed_at = sqlc.arg(closed_at)::timestamptz
WHERE id = sqlc.arg(id);

-- AddCountLine adds a line starting at zero for a variant counted in a full
-- count without having had stock at the start; it does nothing if the line
-- exists.
-- name: AddCountLine :exec
INSERT INTO stock_count_lines (tenant_id, count_id, variant_id, start_quantity, start_cost)
VALUES ($1, $2, $3, 0, 0)
ON CONFLICT DO NOTHING;

-- InsertCountEntry stores the entry in the current round of its line; it
-- inserts nothing if the variant has no line.
-- name: InsertCountEntry :execrows
INSERT INTO stock_count_entries (id, tenant_id, count_id, variant_id, round, quantity, counted_by, counted_at)
SELECT sqlc.arg(id), l.tenant_id, l.count_id, l.variant_id, l.round, sqlc.arg(quantity), sqlc.arg(counted_by), sqlc.arg(counted_at)
FROM stock_count_lines l
WHERE l.count_id = sqlc.arg(count_id) AND l.variant_id = sqlc.arg(variant_id);

-- name: GetCountEntry :one
SELECT id, tenant_id, count_id, variant_id, quantity, counted_by, counted_at
FROM stock_count_entries
WHERE id = $1 AND count_id = $2;

-- name: DeleteCountEntry :execrows
DELETE FROM stock_count_entries WHERE id = $1;

-- name: ListCountEntries :many
SELECT e.id, e.tenant_id, e.count_id, e.variant_id, e.quantity, e.counted_by, e.counted_at
FROM stock_count_entries e
JOIN stock_count_lines l
  ON l.count_id = e.count_id AND l.variant_id = e.variant_id AND l.round = e.round
WHERE e.count_id = sqlc.arg(count_id)
  AND (sqlc.narg(variant_id)::uuid IS NULL OR e.variant_id = sqlc.narg(variant_id))
  AND (sqlc.narg(counted_by)::uuid IS NULL OR e.counted_by = sqlc.narg(counted_by))
ORDER BY e.counted_at DESC, e.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountCountEntries :one
SELECT count(*)
FROM stock_count_entries e
JOIN stock_count_lines l
  ON l.count_id = e.count_id AND l.variant_id = e.variant_id AND l.round = e.round
WHERE e.count_id = sqlc.arg(count_id)
  AND (sqlc.narg(variant_id)::uuid IS NULL OR e.variant_id = sqlc.narg(variant_id))
  AND (sqlc.narg(counted_by)::uuid IS NULL OR e.counted_by = sqlc.narg(counted_by));
