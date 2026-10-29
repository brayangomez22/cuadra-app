-- name: InsertReceivable :exec
INSERT INTO receivables (id, tenant_id, customer_id, sale_id, amount, balance, due_date, status,
                         authorized_by, authorized_at, created_at)
VALUES (sqlc.arg(id), sqlc.arg(tenant_id), sqlc.arg(customer_id), sqlc.arg(sale_id), sqlc.arg(amount),
        sqlc.arg(balance), sqlc.arg(due_date), sqlc.arg(status), sqlc.narg(authorized_by),
        sqlc.narg(authorized_at), sqlc.arg(created_at));

-- name: UpdateReceivable :exec
UPDATE receivables SET balance = $2, status = $3, canceled_at = $4 WHERE id = $1;

-- name: OpenReceivables :many
SELECT * FROM receivables
WHERE customer_id = $1 AND status = 'open'
ORDER BY due_date, created_at, id;

-- name: ReceivableOfSale :one
SELECT id, customer_id, status FROM receivables WHERE sale_id = $1;

-- name: CustomerReceivables :many
SELECT * FROM receivables WHERE customer_id = $1;

-- name: OpenBalances :many
SELECT customer_id, due_date, balance FROM receivables WHERE status = 'open';
