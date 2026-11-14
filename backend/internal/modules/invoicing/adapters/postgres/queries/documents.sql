-- name: AddDocument :exec
INSERT INTO electronic_documents (
    id, tenant_id, sale_id, created_by, kind, resolution_id, prefix, number, issued_at, status, attempts,
    provider_id, code, qr, reasons, last_error, accepted_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18);

-- name: GetDocument :one
SELECT * FROM electronic_documents WHERE tenant_id = $1 AND id = $2;

-- name: SaveDocument :execrows
UPDATE electronic_documents
SET status = $3, attempts = $4, provider_id = $5, code = $6, qr = $7, reasons = $8, last_error = $9,
    accepted_at = $10, updated_at = $11
WHERE tenant_id = $1 AND id = $2;

-- name: ListDocuments :many
SELECT * FROM electronic_documents
WHERE tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
  AND (sqlc.narg(sale_id)::uuid IS NULL OR sale_id = sqlc.narg(sale_id)::uuid)
ORDER BY issued_at DESC, id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountDocuments :one
SELECT count(*) FROM electronic_documents
WHERE tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
  AND (sqlc.narg(sale_id)::uuid IS NULL OR sale_id = sqlc.narg(sale_id)::uuid);

-- name: DocumentExistsForSale :one
SELECT EXISTS (SELECT 1 FROM electronic_documents WHERE tenant_id = $1 AND sale_id = $2);

-- name: SentDocuments :many
SELECT * FROM electronic_documents
WHERE tenant_id = $1 AND status = 'sent'
ORDER BY issued_at, id
LIMIT $2;
