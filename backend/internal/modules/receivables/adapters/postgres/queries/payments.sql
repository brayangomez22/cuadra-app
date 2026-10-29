-- name: InsertCustomerPayment :exec
INSERT INTO customer_payments (id, tenant_id, customer_id, method, amount, user_id, cash_session_id, created_at)
VALUES (sqlc.arg(id), sqlc.arg(tenant_id), sqlc.arg(customer_id), sqlc.arg(method), sqlc.arg(amount),
        sqlc.arg(user_id), sqlc.narg(cash_session_id), sqlc.arg(created_at));

-- name: InsertPaymentAllocation :exec
INSERT INTO customer_payment_allocations (tenant_id, payment_id, receivable_id, position, amount)
VALUES ($1, $2, $3, $4, $5);

-- name: CustomerPayments :many
SELECT id, method, amount, created_at FROM customer_payments WHERE customer_id = $1;
