-- name: CreateCustomer :exec
INSERT INTO customers (id, tenant_id, person_type, document_type, document_number, name, trade_name, email, phone,
                       address, city, segment_id, tax_responsibilities, active, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(tenant_id), sqlc.arg(person_type), sqlc.arg(document_type), sqlc.arg(document_number),
        sqlc.arg(name), sqlc.narg(trade_name), sqlc.narg(email), sqlc.narg(phone), sqlc.narg(address), sqlc.narg(city),
        sqlc.narg(segment_id), sqlc.arg(tax_responsibilities), sqlc.arg(active), sqlc.arg(created_at), sqlc.arg(created_at));

-- name: UpdateCustomer :execrows
UPDATE customers
SET person_type = sqlc.arg(person_type), document_type = sqlc.arg(document_type),
    document_number = sqlc.arg(document_number), name = sqlc.arg(name), trade_name = sqlc.narg(trade_name),
    email = sqlc.narg(email), phone = sqlc.narg(phone), address = sqlc.narg(address), city = sqlc.narg(city),
    segment_id = sqlc.narg(segment_id), tax_responsibilities = sqlc.arg(tax_responsibilities),
    active = sqlc.arg(active), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: GetCustomer :one
SELECT id, tenant_id, person_type, document_type, document_number, name, trade_name, email, phone, address, city,
       segment_id, tax_responsibilities, active, created_at
FROM customers WHERE id = sqlc.arg(id);

-- SearchCustomers matches a document number that starts with the text
-- (without separators), or a name or trade name that has every word of it,
-- ignoring accents and case. An exact document goes first, then the most
-- similar names. Without text it lists by name.
-- name: SearchCustomers :many
SELECT id, tenant_id, person_type, document_type, document_number, name, trade_name, email, phone, address, city,
       segment_id, tax_responsibilities, active, created_at
FROM customers c
WHERE (sqlc.narg(active)::boolean IS NULL OR c.active = sqlc.narg(active))
  AND (sqlc.narg(segment_id)::uuid IS NULL OR c.segment_id = sqlc.narg(segment_id))
  AND (cardinality(sqlc.arg(terms)::text[]) = 0
       OR (sqlc.arg(document)::text <> '' AND replace(c.document_number, '-', '') LIKE sqlc.arg(document)::text || '%')
       OR c.search_name LIKE ALL (SELECT '%' || f_unaccent(lower(term)) || '%' FROM unnest(sqlc.arg(terms)::text[]) AS term))
ORDER BY (sqlc.arg(document)::text <> '' AND replace(c.document_number, '-', '') = sqlc.arg(document)::text) DESC,
         similarity(c.search_name, f_unaccent(lower(sqlc.arg(text)::text))) DESC,
         lower(c.name), c.id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountCustomers :one
SELECT count(*)
FROM customers c
WHERE (sqlc.narg(active)::boolean IS NULL OR c.active = sqlc.narg(active))
  AND (sqlc.narg(segment_id)::uuid IS NULL OR c.segment_id = sqlc.narg(segment_id))
  AND (cardinality(sqlc.arg(terms)::text[]) = 0
       OR (sqlc.arg(document)::text <> '' AND replace(c.document_number, '-', '') LIKE sqlc.arg(document)::text || '%')
       OR c.search_name LIKE ALL (SELECT '%' || f_unaccent(lower(term)) || '%' FROM unnest(sqlc.arg(terms)::text[]) AS term));
