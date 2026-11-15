-- name: PutDocumentFile :exec
INSERT INTO electronic_document_files (tenant_id, document_id, kind, content, created_at)
VALUES ($1, $2, $3, $4, statement_timestamp())
ON CONFLICT (tenant_id, document_id, kind) DO UPDATE
SET content = EXCLUDED.content, created_at = EXCLUDED.created_at;

-- name: GetDocumentFile :one
SELECT content FROM electronic_document_files WHERE tenant_id = $1 AND document_id = $2 AND kind = $3;
