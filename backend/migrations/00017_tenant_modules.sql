-- Modules: the activable modules each tenant has on. The modules themselves
-- (their IDs, dependencies and permissions) are registered in code
-- (internal/platform/modules), so module_id has no foreign key; an ID no
-- longer registered is ignored. A module is active while its row exists:
-- turning it off deletes the row and keeps the module's data.
--
-- Every existing tenant gets the variants module: until now any tenant
-- could add variants, and some already have products with several. New
-- tenants get the modules of their business template (T24g).

-- +goose Up
CREATE TABLE tenant_modules (
    tenant_id  uuid        NOT NULL REFERENCES tenants (id),
    module_id  text        NOT NULL CHECK (module_id ~ '^[a-z][a-z0-9_]{0,63}$'),
    enabled_at timestamptz NOT NULL,
    -- The user who turned it on; NULL when a migration or the sign-up did.
    enabled_by uuid,
    PRIMARY KEY (tenant_id, module_id)
);

ALTER TABLE tenant_modules ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_modules FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON tenant_modules
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- Backfill. FORCE ROW LEVEL SECURITY applies to the owner too, and the owner
-- running migrations on Azure is not a superuser: with no tenant set, the
-- copy would read zero tenants and succeed. FORCE is lifted for the copy
-- and the count check, and restored in the same transaction.
ALTER TABLE tenants NO FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_modules NO FORCE ROW LEVEL SECURITY;

INSERT INTO tenant_modules (tenant_id, module_id, enabled_at, enabled_by)
SELECT id, 'variants', now(), NULL FROM tenants;

-- Checked before FORCE is back: with it, every count would read zero.
-- +goose StatementBegin
DO $$
DECLARE
    tenant_count  bigint;
    with_variants bigint;
BEGIN
    SELECT count(*) INTO tenant_count FROM tenants;
    SELECT count(*) INTO with_variants FROM tenant_modules WHERE module_id = 'variants';
    IF with_variants <> tenant_count THEN
        RAISE EXCEPTION 'tenant_modules backfill: % tenants but % have the variants module', tenant_count, with_variants;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_modules FORCE ROW LEVEL SECURITY;

-- +goose Down
DROP TABLE tenant_modules;
