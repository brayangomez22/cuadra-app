-- name: InsertCashSession :exec
INSERT INTO cash_sessions (id, tenant_id, location_id, user_id, status, opening_float, opened_at)
VALUES ($1, $2, $3, $4, 'open', $5, $6);

-- name: GetCashSession :one
SELECT * FROM cash_sessions WHERE id = $1;

-- LockCashSession reads a session and locks it until the transaction ends:
-- movements and closings of a session run one after the other, and a
-- closing waits for the sales that hold ShareLockCashSession.
-- name: LockCashSession :one
SELECT * FROM cash_sessions WHERE id = $1 FOR UPDATE;

-- ShareLockCashSession reads a session's status and keeps it from closing
-- until the transaction (the one storing a sale) ends. Sales of the same
-- session share the lock and do not wait for each other.
-- name: ShareLockCashSession :one
SELECT status FROM cash_sessions WHERE id = $1 FOR SHARE;

-- name: CurrentCashSession :one
SELECT * FROM cash_sessions WHERE user_id = $1 AND status = 'open';

-- name: InsertCashMovement :exec
INSERT INTO cash_movements (id, tenant_id, session_id, kind, amount, reason, user_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListCashMovements :many
SELECT * FROM cash_movements WHERE session_id = $1 ORDER BY created_at, id;

-- CashSessionSales counts the confirmed sales charged in a session and the
-- change they gave. Voided sales gave their money back: they do not count.
-- name: CashSessionSales :one
SELECT count(*)::integer AS sales_count, coalesce(sum(change), 0)::numeric AS change_given
FROM sales
WHERE cash_session_id = $1 AND status = 'confirmed';

-- CashSessionPayments adds up the payments of the confirmed sales charged in
-- a session by method, in the contract's order of methods.
-- name: CashSessionPayments :many
SELECT p.method, sum(p.amount)::numeric AS amount
FROM sale_payments p
JOIN sales s ON s.tenant_id = p.tenant_id AND s.id = p.sale_id
WHERE s.cash_session_id = $1 AND s.status = 'confirmed'
GROUP BY p.method
ORDER BY array_position(ARRAY['cash', 'card', 'nequi', 'daviplata', 'transfer'], p.method);

-- name: CashSessionMovementTotals :one
SELECT coalesce(sum(amount) FILTER (WHERE kind = 'in'), 0)::numeric  AS cash_in,
       coalesce(sum(amount) FILTER (WHERE kind = 'out'), 0)::numeric AS cash_out
FROM cash_movements
WHERE session_id = $1;

-- name: InsertCashSessionCount :exec
INSERT INTO cash_session_counts (tenant_id, session_id, denomination, quantity)
VALUES ($1, $2, $3, $4);

-- name: InsertCashSessionPayment :exec
INSERT INTO cash_session_payments (tenant_id, session_id, method, amount)
VALUES ($1, $2, $3, $4);

-- name: ListCashSessionCounts :many
SELECT * FROM cash_session_counts WHERE session_id = $1 ORDER BY denomination DESC;

-- name: ListCashSessionPayments :many
SELECT * FROM cash_session_payments WHERE session_id = $1
ORDER BY array_position(ARRAY['cash', 'card', 'nequi', 'daviplata', 'transfer'], method);

-- CloseCashSession stores the closing of an open session; it affects no row
-- if the session is not open.
-- name: CloseCashSession :execrows
UPDATE cash_sessions
SET status = 'closed', closed_at = sqlc.arg(closed_at), closed_by = sqlc.arg(closed_by), notes = sqlc.narg(notes),
    cash_received = sqlc.arg(cash_received), change_given = sqlc.arg(change_given),
    cash_in = sqlc.arg(cash_in), cash_out = sqlc.arg(cash_out), sales_count = sqlc.arg(sales_count),
    expected = sqlc.arg(expected), counted = sqlc.arg(counted), difference = sqlc.arg(difference)
WHERE id = sqlc.arg(id) AND status = 'open';
