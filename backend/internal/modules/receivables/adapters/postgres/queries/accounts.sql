-- name: GetCreditAccount :one
SELECT * FROM credit_accounts WHERE customer_id = $1;

-- LockCreditAccount locks the account until the transaction ends, so the
-- charges, payments and cancellations of a customer run one at a time.
-- name: LockCreditAccount :one
SELECT * FROM credit_accounts WHERE customer_id = $1 FOR UPDATE;

-- name: UpsertCreditTerms :exec
INSERT INTO credit_accounts (tenant_id, customer_id, credit_limit, term_days, updated_at)
VALUES (sqlc.arg(tenant_id), sqlc.arg(customer_id), sqlc.arg(credit_limit), sqlc.arg(term_days), sqlc.arg(updated_at))
ON CONFLICT (tenant_id, customer_id) DO UPDATE
    SET credit_limit = EXCLUDED.credit_limit, term_days = EXCLUDED.term_days, updated_at = EXCLUDED.updated_at;

-- name: SetCreditBalance :exec
UPDATE credit_accounts SET balance = $2, updated_at = now() WHERE customer_id = $1;
