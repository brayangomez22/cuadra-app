-- name: CreateImport :exec
INSERT INTO imports (
    id, tenant_id, created_by, location_id, file_name, format, policy, status,
    total_rows, total_products, errors, file, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
);

-- name: GetImport :one
SELECT id, tenant_id, created_by, location_id, file_name, format, policy, status,
       total_rows, total_products, created, updated, skipped, errors, batches_done,
       created_at, started_at, finished_at
FROM imports
WHERE tenant_id = $1 AND id = $2;

-- name: GetImportFile :one
SELECT file FROM imports WHERE tenant_id = $1 AND id = $2;

-- name: SaveImport :execrows
UPDATE imports
SET status = $3, created = $4, updated = $5, skipped = $6, errors = $7,
    batches_done = $8, started_at = $9, finished_at = $10
WHERE tenant_id = $1 AND id = $2;

-- AddImportErrors inserts many errors in one statement (COPY does not
-- work on a table with row-level security); the arrays have one element
-- per error and unnest walks them together.
-- name: AddImportErrors :exec
INSERT INTO import_errors (tenant_id, import_id, row_number, column_name, code, message)
SELECT sqlc.arg(tenant_id), sqlc.arg(import_id),
       unnest(sqlc.arg(row_numbers)::int[]), unnest(sqlc.arg(column_names)::text[]),
       unnest(sqlc.arg(codes)::text[]), unnest(sqlc.arg(messages)::text[]);

-- name: ListImportErrors :many
SELECT row_number, column_name, code, message
FROM import_errors
WHERE tenant_id = $1 AND import_id = $2
ORDER BY row_number, id
LIMIT $3 OFFSET $4;

-- name: CountImportErrors :one
SELECT count(*) FROM import_errors WHERE tenant_id = $1 AND import_id = $2;
