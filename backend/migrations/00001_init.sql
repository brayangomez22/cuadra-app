-- Initial setup: extensions and the application role.
--
-- Migrations run as the schema owner. The API connects as app_user, which is
-- neither a superuser nor BYPASSRLS nor the owner of any table, so Row-Level
-- Security always applies to it. Its password is not set here: cmd/migrate sets
-- it from APP_DB_PASSWORD (Key Vault in Azure).

-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Roles are cluster-wide: another database on the same server may have created it.
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        CREATE ROLE app_user LOGIN;
    END IF;
END
$$;
-- +goose StatementEnd

-- Enforced even if the role already existed with other attributes.
ALTER ROLE app_user LOGIN NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE NOREPLICATION NOINHERIT;

-- +goose StatementBegin
DO $$
BEGIN
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO app_user', current_database());
END
$$;
-- +goose StatementEnd

-- Data access only: no CREATE on the schema and no TRUNCATE (it bypasses RLS).
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO app_user;
-- Applies to the tables and sequences that later migrations create (as the
-- role running them).
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO app_user;

-- +goose Down
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE USAGE, SELECT ON SEQUENCES FROM app_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM app_user;
REVOKE USAGE ON SCHEMA public FROM app_user;

-- +goose StatementBegin
DO $$
BEGIN
    EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM app_user', current_database());
END
$$;
-- +goose StatementEnd

-- The role is kept if it still has privileges in another database of the cluster.
-- +goose StatementBegin
DO $$
BEGIN
    DROP ROLE IF EXISTS app_user;
EXCEPTION WHEN dependent_objects_still_exist THEN
    RAISE NOTICE 'app_user still has privileges in another database; not dropped';
END
$$;
-- +goose StatementEnd

DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS pgcrypto;
