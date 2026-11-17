-- name: ListGlobalFlags :many
SELECT key, enabled FROM feature_flags;

-- name: SetGlobalFlag :exec
INSERT INTO feature_flags (key, enabled, updated_at, updated_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (key) DO UPDATE
SET enabled = EXCLUDED.enabled, updated_at = EXCLUDED.updated_at, updated_by = EXCLUDED.updated_by;

-- name: ListTenantFlags :many
SELECT key, enabled FROM tenant_feature_flags WHERE tenant_id = $1;

-- name: SetTenantFlag :exec
INSERT INTO tenant_feature_flags (tenant_id, key, enabled, updated_at, updated_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tenant_id, key) DO UPDATE
SET enabled = EXCLUDED.enabled, updated_at = EXCLUDED.updated_at, updated_by = EXCLUDED.updated_by;

-- name: ClearTenantFlag :exec
DELETE FROM tenant_feature_flags WHERE tenant_id = $1 AND key = $2;
