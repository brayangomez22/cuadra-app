-- Feature flags (T39): a global value per flag and exceptions per tenant.
-- The flags themselves are declared in code (platform/flags); these tables
-- only keep what the platform's operators changed. A flag with no row is off.
--
-- feature_flags is platform configuration, not business data: it has no
-- tenant_id and no RLS. tenant_feature_flags is per tenant, with RLS, so the
-- export and the deletion of a tenant (T49) find its exceptions.
-- The operator screen lists the exceptions of every tenant through
-- feature_flag_exceptions(), which runs as flag_lookup.

-- +goose Up
CREATE TABLE feature_flags (
    key        text        PRIMARY KEY CHECK (key ~ '^[a-z][a-z0-9_]{0,63}$'),
    enabled    boolean     NOT NULL,
    updated_at timestamptz NOT NULL,
    -- The operator who changed it last.
    updated_by uuid        NOT NULL
);

CREATE TABLE tenant_feature_flags (
    tenant_id  uuid        NOT NULL REFERENCES tenants (id),
    key        text        NOT NULL CHECK (key ~ '^[a-z][a-z0-9_]{0,63}$'),
    enabled    boolean     NOT NULL,
    updated_at timestamptz NOT NULL,
    updated_by uuid        NOT NULL,
    PRIMARY KEY (tenant_id, key)
);

ALTER TABLE tenant_feature_flags ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_feature_flags FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tenant_feature_flags
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'flag_lookup') THEN
        CREATE ROLE flag_lookup NOLOGIN;
    END IF;
    -- The migration role must be a member to make flag_lookup the
    -- function's owner (not implied by CREATEROLE since PostgreSQL 16).
    EXECUTE format('GRANT flag_lookup TO %I', current_user);
END
$$;
-- +goose StatementEnd

-- Only a superuser may set SUPERUSER, BYPASSRLS and REPLICATION, even to turn
-- them off. The migration role on Azure is not one: there they are verified.
-- +goose StatementBegin
DO $$
BEGIN
    ALTER ROLE flag_lookup NOLOGIN NOCREATEDB NOCREATEROLE NOINHERIT;
    IF (SELECT rolsuper FROM pg_roles WHERE rolname = current_user) THEN
        ALTER ROLE flag_lookup NOSUPERUSER NOBYPASSRLS NOREPLICATION;
    ELSIF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'flag_lookup' AND (rolsuper OR rolbypassrls OR rolreplication)) THEN
        RAISE EXCEPTION 'flag_lookup is a superuser or has BYPASSRLS or REPLICATION: remove them as a superuser';
    END IF;
END
$$;
-- +goose StatementEnd

GRANT USAGE ON SCHEMA public TO flag_lookup;
GRANT SELECT (tenant_id, key, enabled, updated_at) ON tenant_feature_flags TO flag_lookup;
GRANT SELECT (id, name) ON tenants TO flag_lookup;

-- RLS still applies to flag_lookup (FORCE): these policies let it read every
-- row, but only through the columns granted above.
CREATE POLICY flag_lookup_read ON tenant_feature_flags
    FOR SELECT TO flag_lookup
    USING (true);
CREATE POLICY flag_lookup_read ON tenants
    FOR SELECT TO flag_lookup
    USING (true);

CREATE FUNCTION feature_flag_exceptions()
    RETURNS TABLE (tenant_id uuid, tenant_name text, key text, enabled boolean, updated_at timestamptz)
    LANGUAGE sql STABLE SECURITY DEFINER
    SET search_path = pg_catalog, pg_temp
    AS $$
        SELECT f.tenant_id, t.name, f.key, f.enabled, f.updated_at
        FROM public.tenant_feature_flags f
        JOIN public.tenants t ON t.id = f.tenant_id
        ORDER BY f.key, t.name, f.tenant_id
    $$;

-- A new owner needs CREATE on the schema, which 00001 took from PUBLIC; a
-- superuser skips that check, the migration role on Azure does not.
GRANT CREATE ON SCHEMA public TO flag_lookup;
ALTER FUNCTION feature_flag_exceptions() OWNER TO flag_lookup;
REVOKE CREATE ON SCHEMA public FROM flag_lookup;
REVOKE ALL ON FUNCTION feature_flag_exceptions() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION feature_flag_exceptions() TO app_user;

-- +goose Down
DROP FUNCTION feature_flag_exceptions();
DROP POLICY flag_lookup_read ON tenants;
REVOKE SELECT (id, name) ON tenants FROM flag_lookup;
DROP TABLE tenant_feature_flags;
REVOKE USAGE ON SCHEMA public FROM flag_lookup;
DROP TABLE feature_flags;
