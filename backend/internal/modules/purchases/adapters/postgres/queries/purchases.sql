-- name: InsertPurchase :exec
INSERT INTO purchases (id, tenant_id, supplier_id, location_id, invoice_number, invoice_key, issued_on, due_on,
                       payment, notes, subtotal, tax, total, balance_due, created_by, created_at)
VALUES (sqlc.arg(id), sqlc.arg(tenant_id), sqlc.arg(supplier_id), sqlc.arg(location_id), sqlc.arg(invoice_number),
        sqlc.arg(invoice_key), sqlc.arg(issued_on), sqlc.narg(due_on), sqlc.arg(payment), sqlc.narg(notes),
        sqlc.arg(subtotal), sqlc.arg(tax), sqlc.arg(total), sqlc.arg(balance_due), sqlc.arg(created_by),
        sqlc.arg(created_at));

-- name: InsertPurchaseLine :exec
INSERT INTO purchase_lines (tenant_id, purchase_id, position, variant_id, product_name, variant_label, unit_id,
                            unit_code, unit_factor, quantity, unit_cost, tax_rate, subtotal, tax, base_quantity,
                            base_unit_cost)
VALUES (sqlc.arg(tenant_id), sqlc.arg(purchase_id), sqlc.arg(position), sqlc.arg(variant_id), sqlc.arg(product_name),
        sqlc.narg(variant_label), sqlc.narg(unit_id), sqlc.narg(unit_code), sqlc.arg(unit_factor), sqlc.arg(quantity),
        sqlc.arg(unit_cost), sqlc.arg(tax_rate), sqlc.arg(subtotal), sqlc.arg(tax), sqlc.arg(base_quantity),
        sqlc.arg(base_unit_cost));

-- name: GetPurchase :one
SELECT id, tenant_id, supplier_id, location_id, invoice_number, issued_on, due_on, payment, notes, balance_due,
       voided_at, voided_by, void_reason, created_by, created_at
FROM purchases WHERE id = sqlc.arg(id);

-- name: LockPurchase :one
SELECT id FROM purchases WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: UpdatePurchaseVoid :exec
UPDATE purchases
SET voided_at = sqlc.arg(voided_at), voided_by = sqlc.arg(voided_by), void_reason = sqlc.arg(void_reason),
    balance_due = sqlc.arg(balance_due)
WHERE id = sqlc.arg(id);

-- name: UpdatePurchaseBalance :exec
UPDATE purchases SET balance_due = sqlc.arg(balance_due) WHERE id = sqlc.arg(id);

-- name: InsertSupplierPayment :exec
INSERT INTO supplier_payments (id, tenant_id, purchase_id, supplier_id, method, amount, balance_after, paid_on,
                               created_by, created_at)
VALUES (sqlc.arg(id), sqlc.arg(tenant_id), sqlc.arg(purchase_id), sqlc.arg(supplier_id), sqlc.arg(method),
        sqlc.arg(amount), sqlc.arg(balance_after), sqlc.arg(paid_on), sqlc.arg(created_by), sqlc.arg(created_at));

-- name: SupplierPayments :many
SELECT id, tenant_id, purchase_id, supplier_id, method, amount, balance_after, paid_on, created_by, created_at
FROM supplier_payments WHERE purchase_id = sqlc.arg(purchase_id) ORDER BY created_at, id;

-- name: Payables :many
-- What the tenant owes each supplier: its open purchases (a voided one owes
-- nothing), split by due date. The one with the most overdue goes first.
SELECT p.supplier_id, s.name AS supplier_name,
       sum(p.balance_due)::numeric(18,4) AS balance,
       coalesce(sum(p.balance_due) FILTER (WHERE p.due_on < sqlc.arg(overdue_before)::date), 0)::numeric(18,4) AS overdue,
       coalesce(sum(p.balance_due) FILTER (WHERE p.due_on >= sqlc.arg(overdue_before)::date
                                             AND p.due_on <= sqlc.arg(due_soon_until)::date), 0)::numeric(18,4) AS due_soon,
       count(*) AS purchases,
       min(p.due_on)::date AS next_due
FROM purchases p
JOIN suppliers s ON s.tenant_id = p.tenant_id AND s.id = p.supplier_id
WHERE p.balance_due > 0
GROUP BY p.supplier_id, s.name
ORDER BY overdue DESC, balance DESC, s.name, p.supplier_id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: PayablesTotals :one
SELECT coalesce(sum(balance_due), 0)::numeric(18,4) AS balance,
       coalesce(sum(balance_due) FILTER (WHERE due_on < sqlc.arg(overdue_before)::date), 0)::numeric(18,4) AS overdue,
       coalesce(sum(balance_due) FILTER (WHERE due_on >= sqlc.arg(overdue_before)::date
                                           AND due_on <= sqlc.arg(due_soon_until)::date), 0)::numeric(18,4) AS due_soon,
       count(DISTINCT supplier_id) AS suppliers
FROM purchases
WHERE balance_due > 0;

-- name: PurchaseLines :many
SELECT variant_id, product_name, variant_label, unit_id, unit_code, unit_factor, quantity, unit_cost, tax_rate,
       base_quantity, base_unit_cost
FROM purchase_lines WHERE purchase_id = sqlc.arg(purchase_id) ORDER BY position;

-- name: SearchPurchases :many
SELECT p.id, p.supplier_id, s.name AS supplier_name, p.location_id, p.invoice_number, p.issued_on, p.due_on,
       p.payment, p.total, p.balance_due, (p.voided_at IS NOT NULL)::boolean AS voided, p.created_at,
       (SELECT count(*) FROM purchase_lines l WHERE l.tenant_id = p.tenant_id AND l.purchase_id = p.id) AS line_count
FROM purchases p
JOIN suppliers s ON s.tenant_id = p.tenant_id AND s.id = p.supplier_id
WHERE (sqlc.narg(supplier_id)::uuid IS NULL OR p.supplier_id = sqlc.narg(supplier_id))
  AND (sqlc.narg(status)::text IS NULL OR sqlc.narg(status) = CASE
        WHEN p.voided_at IS NOT NULL THEN 'voided' WHEN p.balance_due > 0 THEN 'open' ELSE 'paid' END)
  AND (sqlc.narg(issued_from)::date IS NULL OR p.issued_on >= sqlc.narg(issued_from))
  AND (sqlc.narg(issued_to)::date IS NULL OR p.issued_on <= sqlc.narg(issued_to))
ORDER BY p.issued_on DESC, p.created_at DESC, p.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountPurchases :one
SELECT count(*)
FROM purchases p
WHERE (sqlc.narg(supplier_id)::uuid IS NULL OR p.supplier_id = sqlc.narg(supplier_id))
  AND (sqlc.narg(status)::text IS NULL OR sqlc.narg(status) = CASE
        WHEN p.voided_at IS NOT NULL THEN 'voided' WHEN p.balance_due > 0 THEN 'open' ELSE 'paid' END)
  AND (sqlc.narg(issued_from)::date IS NULL OR p.issued_on >= sqlc.narg(issued_from))
  AND (sqlc.narg(issued_to)::date IS NULL OR p.issued_on <= sqlc.narg(issued_to));

-- name: SupplierBalances :many
SELECT supplier_id, sum(balance_due)::numeric(18,4) AS balance_due
FROM purchases
WHERE supplier_id = ANY(sqlc.arg(ids)::uuid[]) AND balance_due > 0
GROUP BY supplier_id;
