-- NextSaleNumber draws the next number of a location. The upsert locks the
-- sequence row until the transaction ends: concurrent sales of the location
-- wait, and a rollback undoes the increment, so numbers have no gaps.
-- name: NextSaleNumber :one
INSERT INTO sale_sequences (tenant_id, location_id, last_number)
VALUES (sqlc.arg(tenant_id), sqlc.arg(location_id), 1)
ON CONFLICT (tenant_id, location_id) DO UPDATE SET last_number = sale_sequences.last_number + 1
RETURNING last_number;

-- name: InsertSale :exec
INSERT INTO sales (id, tenant_id, location_id, number, user_id, customer_id, status,
                   subtotal, discounts, tax, total, paid, change, created_at, confirmed_at)
VALUES (sqlc.arg(id), sqlc.arg(tenant_id), sqlc.arg(location_id), sqlc.arg(number), sqlc.arg(user_id),
        sqlc.narg(customer_id), sqlc.arg(status), sqlc.arg(subtotal), sqlc.arg(discounts), sqlc.arg(tax),
        sqlc.arg(total), sqlc.arg(paid), sqlc.arg(change), sqlc.arg(created_at), sqlc.arg(confirmed_at));

-- name: InsertSaleLine :exec
INSERT INTO sale_lines (tenant_id, sale_id, position, product_id, product_name, quantity, unit_price,
                        tax_rate, discount, gross, tax)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: InsertSalePayment :exec
INSERT INTO sale_payments (tenant_id, sale_id, position, method, amount)
VALUES ($1, $2, $3, $4, $5);

-- name: GetSale :one
SELECT * FROM sales WHERE id = $1;

-- LockSale reads a sale and locks it until the transaction ends, so two
-- voids of the same sale run one after the other.
-- name: LockSale :one
SELECT * FROM sales WHERE id = $1 FOR UPDATE;

-- name: ListSaleLines :many
SELECT * FROM sale_lines WHERE sale_id = $1 ORDER BY position;

-- name: ListSalePayments :many
SELECT * FROM sale_payments WHERE sale_id = $1 ORDER BY position;

-- VoidSale stores the void of a confirmed sale; it affects no row if the
-- sale is not confirmed.
-- name: VoidSale :execrows
UPDATE sales
SET status = 'voided', voided_at = sqlc.arg(voided_at), voided_by = sqlc.arg(voided_by), void_reason = sqlc.arg(void_reason)
WHERE id = sqlc.arg(id) AND status = 'confirmed';

-- ListSales pages the sales confirmed in [from, to), newest first.
-- name: ListSales :many
SELECT id, number, location_id, user_id, status, total, confirmed_at, voided_at
FROM sales
WHERE confirmed_at >= sqlc.arg(from_time) AND confirmed_at < sqlc.arg(to_time)
  AND (sqlc.narg(location_id)::uuid IS NULL OR location_id = sqlc.narg(location_id))
ORDER BY confirmed_at DESC, id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountSales :one
SELECT count(*)
FROM sales
WHERE confirmed_at >= sqlc.arg(from_time) AND confirmed_at < sqlc.arg(to_time)
  AND (sqlc.narg(location_id)::uuid IS NULL OR location_id = sqlc.narg(location_id));
