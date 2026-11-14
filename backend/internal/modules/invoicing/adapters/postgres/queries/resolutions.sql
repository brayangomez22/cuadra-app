-- name: CreateResolution :exec
INSERT INTO numbering_resolutions (
    id, tenant_id, kind, prefix, number, technical_key, range_from, range_to, valid_from, valid_to, next_number, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: ListResolutions :many
SELECT id, tenant_id, kind, prefix, number, technical_key, range_from, range_to, valid_from, valid_to, next_number, created_at
FROM numbering_resolutions
WHERE tenant_id = $1
ORDER BY created_at DESC, id DESC;

-- name: LockResolutionsOfKind :many
SELECT id, tenant_id, kind, prefix, number, technical_key, range_from, range_to, valid_from, valid_to, next_number, created_at
FROM numbering_resolutions
WHERE tenant_id = $1 AND kind = $2
ORDER BY id
FOR UPDATE;

-- name: SaveResolutionNext :execrows
UPDATE numbering_resolutions SET next_number = $3 WHERE tenant_id = $1 AND id = $2;

-- name: DeleteUnusedResolution :execrows
DELETE FROM numbering_resolutions
WHERE tenant_id = $1 AND id = $2 AND next_number = range_from;

-- name: ResolutionExists :one
SELECT EXISTS (SELECT 1 FROM numbering_resolutions WHERE tenant_id = $1 AND id = $2);
