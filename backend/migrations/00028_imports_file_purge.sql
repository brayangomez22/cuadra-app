-- The file of an import is deleted 30 days after it ends (P82, T34): the
-- import and its errors stay. A daily job finds them across tenants with
-- purgeable_import_files(), which runs as import_lookup and returns ids
-- only, and deletes each file within its tenant.

-- +goose Up
ALTER TABLE imports ALTER COLUMN file DROP NOT NULL;
ALTER TABLE imports ADD COLUMN file_purged_at timestamptz;
ALTER TABLE imports ADD CONSTRAINT imports_file_purged_check
    CHECK ((file IS NULL) = (file_purged_at IS NOT NULL));

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'import_lookup') THEN
        CREATE ROLE import_lookup NOLOGIN;
    END IF;
    -- The migration role must be a member to make import_lookup the
    -- function's owner (not implied by CREATEROLE since PostgreSQL 16).
    EXECUTE format('GRANT import_lookup TO %I', current_user);
END
$$;
-- +goose StatementEnd

-- Only a superuser may set SUPERUSER, BYPASSRLS and REPLICATION, even to turn
-- them off. The migration role on Azure is not one: there they are verified.
-- +goose StatementBegin
DO $$
BEGIN
    ALTER ROLE import_lookup NOLOGIN NOCREATEDB NOCREATEROLE NOINHERIT;
    IF (SELECT rolsuper FROM pg_roles WHERE rolname = current_user) THEN
        ALTER ROLE import_lookup NOSUPERUSER NOBYPASSRLS NOREPLICATION;
    ELSIF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'import_lookup' AND (rolsuper OR rolbypassrls OR rolreplication)) THEN
        RAISE EXCEPTION 'import_lookup is a superuser or has BYPASSRLS or REPLICATION: remove them as a superuser';
    END IF;
END
$$;
-- +goose StatementEnd

GRANT USAGE ON SCHEMA public TO import_lookup;
GRANT SELECT (id, tenant_id, status, created_at, finished_at, file_purged_at) ON imports TO import_lookup;

-- RLS still applies to import_lookup (FORCE): this policy lets it read every
-- row, but only through the columns granted above (never the file).
CREATE POLICY file_purge_lookup ON imports
    FOR SELECT TO import_lookup
    USING (true);

CREATE FUNCTION purgeable_import_files(p_before timestamptz, p_statuses text[], p_limit int)
    RETURNS TABLE (tenant_id uuid, import_id uuid)
    LANGUAGE sql STABLE SECURITY DEFINER
    SET search_path = pg_catalog, pg_temp
    AS $$
        SELECT i.tenant_id, i.id
        FROM public.imports i
        WHERE i.file_purged_at IS NULL
          AND i.status = ANY (p_statuses)
          AND coalesce(i.finished_at, i.created_at) < p_before
        ORDER BY i.id
        LIMIT p_limit
    $$;

-- A new owner needs CREATE on the schema, which 00001 took from PUBLIC; a
-- superuser skips that check, the migration role on Azure does not.
GRANT CREATE ON SCHEMA public TO import_lookup;
ALTER FUNCTION purgeable_import_files(timestamptz, text[], int) OWNER TO import_lookup;
REVOKE CREATE ON SCHEMA public FROM import_lookup;
REVOKE ALL ON FUNCTION purgeable_import_files(timestamptz, text[], int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION purgeable_import_files(timestamptz, text[], int) TO app_user;

-- +goose Down
DROP FUNCTION purgeable_import_files(timestamptz, text[], int);
DROP POLICY file_purge_lookup ON imports;
REVOKE SELECT (id, tenant_id, status, created_at, finished_at, file_purged_at) ON imports FROM import_lookup;
REVOKE USAGE ON SCHEMA public FROM import_lookup;
ALTER TABLE imports DROP CONSTRAINT imports_file_purged_check;
-- The owner sees every tenant's imports only without FORCE.
ALTER TABLE imports NO FORCE ROW LEVEL SECURITY;
UPDATE imports SET file = ''::bytea WHERE file IS NULL;
ALTER TABLE imports FORCE ROW LEVEL SECURITY;
ALTER TABLE imports DROP COLUMN file_purged_at;
ALTER TABLE imports ALTER COLUMN file SET NOT NULL;
