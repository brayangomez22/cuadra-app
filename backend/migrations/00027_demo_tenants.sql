-- Demonstration tenants (T34): the public demo is a tenant marked demo,
-- generated with realistic data and reset every night. Resetting deletes it
-- with all its rows, which app_user cannot do (the kardex, the sales and the
-- cash sessions revoke DELETE from it): purge_demo_tenant() does it as the
-- demo_purge role, and only for a tenant marked demo.

-- +goose Up
ALTER TABLE tenants ADD COLUMN demo boolean NOT NULL DEFAULT false;

CREATE INDEX tenants_demo_idx ON tenants (business_template) WHERE demo;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'demo_purge') THEN
        CREATE ROLE demo_purge NOLOGIN;
    END IF;
    -- The migration role must be a member to make demo_purge the functions'
    -- owner (not implied by CREATEROLE since PostgreSQL 16, e.g. on Azure).
    EXECUTE format('GRANT demo_purge TO %I', current_user);
END
$$;
-- +goose StatementEnd

-- Only a superuser may set SUPERUSER, BYPASSRLS and REPLICATION, even to turn
-- them off. The migration role on Azure is not one: there they are verified.
-- +goose StatementBegin
DO $$
BEGIN
    ALTER ROLE demo_purge NOLOGIN NOCREATEDB NOCREATEROLE NOINHERIT;
    IF (SELECT rolsuper FROM pg_roles WHERE rolname = current_user) THEN
        ALTER ROLE demo_purge NOSUPERUSER NOBYPASSRLS NOREPLICATION;
    ELSIF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'demo_purge' AND (rolsuper OR rolbypassrls OR rolreplication)) THEN
        RAISE EXCEPTION 'demo_purge is a superuser or has BYPASSRLS or REPLICATION: remove them as a superuser';
    END IF;
END
$$;
-- +goose StatementEnd

-- demo_purge reads and deletes the rows of every business table. RLS still
-- applies to it (FORCE): the functions scope it to one tenant with
-- app.tenant_id, as any tenant transaction. The tables of later migrations
-- get the same privileges by default; a test checks every table has them.
GRANT USAGE ON SCHEMA public TO demo_purge;
-- +goose StatementBegin
DO $$
DECLARE
    t text;
BEGIN
    FOR t IN
        SELECT c.relname FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_attribute a ON a.attrelid = c.oid
        WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p') AND a.attname = 'tenant_id' AND NOT a.attisdropped
    LOOP
        EXECUTE format('GRANT SELECT, DELETE ON public.%I TO demo_purge', t);
    END LOOP;
END
$$;
-- +goose StatementEnd
GRANT SELECT, DELETE ON tenants TO demo_purge;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, DELETE ON TABLES TO demo_purge;

-- Finding the demo tenants crosses tenants (the reset runs with none): this
-- policy lets demo_purge see the demo rows of tenants, and nothing else.
CREATE POLICY demo_lookup ON tenants
    FOR SELECT TO demo_purge
    USING (demo);

CREATE FUNCTION demo_tenants(p_template text)
    RETURNS SETOF uuid
    LANGUAGE sql STABLE SECURITY DEFINER
    SET search_path = pg_catalog, pg_temp
    AS $$
        SELECT t.id FROM public.tenants t
        WHERE t.demo AND t.business_template = p_template
        ORDER BY t.id
    $$;

-- Deletes a demo tenant and every row with its tenant_id, in an order the
-- foreign keys accept: each pass deletes the tables nothing left references
-- and retries the rest, so a new table needs no change here. Returns
-- 'purged', 'not_found' or 'not_demo' (and then deletes nothing).
-- +goose StatementBegin
CREATE FUNCTION purge_demo_tenant(p_tenant uuid)
    RETURNS text
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path = pg_catalog, pg_temp
    AS $$
DECLARE
    v_demo     boolean;
    v_pending  text[];
    v_left     text[];
    v_table    text;
    v_progress boolean;
BEGIN
    PERFORM set_config('app.tenant_id', p_tenant::text, true);
    SELECT t.demo INTO v_demo FROM public.tenants t WHERE t.id = p_tenant;
    IF NOT FOUND THEN
        RETURN 'not_found';
    END IF;
    IF NOT v_demo THEN
        RETURN 'not_demo';
    END IF;

    SELECT array_agg(c.relname::text ORDER BY c.relname) INTO v_pending
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN pg_attribute a ON a.attrelid = c.oid
    WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p') AND a.attname = 'tenant_id' AND NOT a.attisdropped;

    LOOP
        v_left := '{}';
        v_progress := false;
        FOREACH v_table IN ARRAY v_pending LOOP
            BEGIN
                EXECUTE format('DELETE FROM public.%I WHERE tenant_id = $1', v_table) USING p_tenant;
                v_progress := true;
            EXCEPTION WHEN foreign_key_violation THEN
                v_left := v_left || v_table;
            END;
        END LOOP;
        EXIT WHEN cardinality(v_left) = 0;
        IF NOT v_progress THEN
            RAISE EXCEPTION 'purge_demo_tenant: cannot delete the rows of %', v_left;
        END IF;
        v_pending := v_left;
    END LOOP;

    DELETE FROM public.tenants WHERE id = p_tenant;
    RETURN 'purged';
END
$$;
-- +goose StatementEnd

-- A new owner needs CREATE on the schema, which 00001 took from PUBLIC; a
-- superuser skips that check, the migration role on Azure does not.
GRANT CREATE ON SCHEMA public TO demo_purge;
ALTER FUNCTION demo_tenants(text) OWNER TO demo_purge;
ALTER FUNCTION purge_demo_tenant(uuid) OWNER TO demo_purge;
REVOKE CREATE ON SCHEMA public FROM demo_purge;
REVOKE ALL ON FUNCTION demo_tenants(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION purge_demo_tenant(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION demo_tenants(text) TO app_user;
GRANT EXECUTE ON FUNCTION purge_demo_tenant(uuid) TO app_user;

-- +goose Down
DROP FUNCTION purge_demo_tenant(uuid);
DROP FUNCTION demo_tenants(text);
DROP POLICY demo_lookup ON tenants;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT, DELETE ON TABLES FROM demo_purge;
-- +goose StatementBegin
DO $$
DECLARE
    t text;
BEGIN
    FOR t IN
        SELECT c.relname FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p')
    LOOP
        EXECUTE format('REVOKE ALL ON public.%I FROM demo_purge', t);
    END LOOP;
END
$$;
-- +goose StatementEnd
REVOKE USAGE ON SCHEMA public FROM demo_purge;
DROP INDEX tenants_demo_idx;
ALTER TABLE tenants DROP COLUMN demo;
