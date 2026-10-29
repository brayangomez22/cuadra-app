-- name: CreateSupplier :exec
INSERT INTO suppliers (id, tenant_id, name, nit, contact_name, email, phone, payment_term_days, active, created_at,
                       updated_at)
VALUES (sqlc.arg(id), sqlc.arg(tenant_id), sqlc.arg(name), sqlc.arg(nit), sqlc.narg(contact_name), sqlc.narg(email),
        sqlc.narg(phone), sqlc.arg(payment_term_days), sqlc.arg(active), sqlc.arg(created_at), sqlc.arg(created_at));

-- name: UpdateSupplier :execrows
UPDATE suppliers
SET name = sqlc.arg(name), nit = sqlc.arg(nit), contact_name = sqlc.narg(contact_name), email = sqlc.narg(email),
    phone = sqlc.narg(phone), payment_term_days = sqlc.arg(payment_term_days), active = sqlc.arg(active),
    updated_at = now()
WHERE id = sqlc.arg(id);

-- name: GetSupplier :one
SELECT id, tenant_id, name, nit, contact_name, email, phone, payment_term_days, active, created_at
FROM suppliers WHERE id = sqlc.arg(id);

-- SearchSuppliers matches a NIT whose number starts with the text (without
-- separators), or a name that has every word of it, ignoring accents and
-- case. It lists by name.
-- name: SearchSuppliers :many
SELECT id, tenant_id, name, nit, contact_name, email, phone, payment_term_days, active, created_at
FROM suppliers s
WHERE (sqlc.narg(active)::boolean IS NULL OR s.active = sqlc.narg(active))
  AND (cardinality(sqlc.arg(terms)::text[]) = 0
       OR (sqlc.arg(nit)::text <> '' AND s.nit LIKE sqlc.arg(nit)::text || '%')
       OR s.search_name LIKE ALL (SELECT '%' || f_unaccent(lower(term)) || '%' FROM unnest(sqlc.arg(terms)::text[]) AS term))
ORDER BY lower(s.name), s.id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountSuppliers :one
SELECT count(*)
FROM suppliers s
WHERE (sqlc.narg(active)::boolean IS NULL OR s.active = sqlc.narg(active))
  AND (cardinality(sqlc.arg(terms)::text[]) = 0
       OR (sqlc.arg(nit)::text <> '' AND s.nit LIKE sqlc.arg(nit)::text || '%')
       OR s.search_name LIKE ALL (SELECT '%' || f_unaccent(lower(term)) || '%' FROM unnest(sqlc.arg(terms)::text[]) AS term));
