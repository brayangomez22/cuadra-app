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
       created_by, created_at
FROM purchases WHERE id = sqlc.arg(id);

-- name: PurchaseLines :many
SELECT variant_id, product_name, variant_label, unit_id, unit_code, unit_factor, quantity, unit_cost, tax_rate,
       base_quantity, base_unit_cost
FROM purchase_lines WHERE purchase_id = sqlc.arg(purchase_id) ORDER BY position;

-- name: SearchPurchases :many
SELECT p.id, p.supplier_id, s.name AS supplier_name, p.location_id, p.invoice_number, p.issued_on, p.due_on,
       p.payment, p.total, p.balance_due, p.created_at,
       (SELECT count(*) FROM purchase_lines l WHERE l.tenant_id = p.tenant_id AND l.purchase_id = p.id) AS line_count
FROM purchases p
JOIN suppliers s ON s.tenant_id = p.tenant_id AND s.id = p.supplier_id
WHERE (sqlc.narg(supplier_id)::uuid IS NULL OR p.supplier_id = sqlc.narg(supplier_id))
  AND (sqlc.narg(open)::boolean IS NULL OR (p.balance_due > 0) = sqlc.narg(open))
  AND (sqlc.narg(issued_from)::date IS NULL OR p.issued_on >= sqlc.narg(issued_from))
  AND (sqlc.narg(issued_to)::date IS NULL OR p.issued_on <= sqlc.narg(issued_to))
ORDER BY p.issued_on DESC, p.created_at DESC, p.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountPurchases :one
SELECT count(*)
FROM purchases p
WHERE (sqlc.narg(supplier_id)::uuid IS NULL OR p.supplier_id = sqlc.narg(supplier_id))
  AND (sqlc.narg(open)::boolean IS NULL OR (p.balance_due > 0) = sqlc.narg(open))
  AND (sqlc.narg(issued_from)::date IS NULL OR p.issued_on >= sqlc.narg(issued_from))
  AND (sqlc.narg(issued_to)::date IS NULL OR p.issued_on <= sqlc.narg(issued_to));

-- name: SupplierBalances :many
SELECT supplier_id, sum(balance_due)::numeric(18,4) AS balance_due
FROM purchases
WHERE supplier_id = ANY(sqlc.arg(ids)::uuid[]) AND balance_due > 0
GROUP BY supplier_id;
