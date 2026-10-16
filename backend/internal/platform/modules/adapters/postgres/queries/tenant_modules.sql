-- name: ListActiveModules :many
SELECT module_id FROM tenant_modules WHERE tenant_id = $1;

-- name: LockTenantModules :exec
-- Serializes the module changes of one tenant until the transaction ends.
SELECT pg_advisory_xact_lock(hashtextextended('tenant_modules:' || sqlc.arg(tenant_id)::uuid::text, 0));

-- name: EnableModule :exec
INSERT INTO tenant_modules (tenant_id, module_id, enabled_at, enabled_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id, module_id) DO NOTHING;

-- name: DisableModule :exec
DELETE FROM tenant_modules WHERE tenant_id = $1 AND module_id = $2;
