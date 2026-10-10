-- Authentication: refresh tokens, and the login lookup by email across tenants.

-- +goose Up

-- Only the SHA-256 of each token's secret is stored. A family is the chain of
-- rotations started by one login; reusing a revoked token revokes the family.
CREATE TABLE refresh_tokens (
    id         uuid        PRIMARY KEY,
    tenant_id  uuid        NOT NULL REFERENCES tenants (id),
    user_id    uuid        NOT NULL REFERENCES users (id),
    family_id  uuid        NOT NULL,
    token_hash bytea       NOT NULL CHECK (octet_length(token_hash) = 32),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    revoked_at timestamptz,
    CONSTRAINT refresh_tokens_token_hash_key UNIQUE (token_hash)
);

CREATE INDEX refresh_tokens_family_idx ON refresh_tokens (tenant_id, family_id);

ALTER TABLE refresh_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE refresh_tokens FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON refresh_tokens
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- Login lookup. An email is unique per tenant, not globally, and the login
-- runs before the tenant is known, so the app (subject to RLS) cannot find
-- the user by itself. login_candidates() answers only "which (tenant, user)
-- pairs have this exact email": it runs as auth_lookup, a role that cannot
-- log in, that app_user cannot become, and that may read only the id,
-- tenant_id and email columns of users. Nothing else bypasses RLS.

-- Roles are cluster-wide: another database on the same server may have created it.
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'auth_lookup') THEN
        CREATE ROLE auth_lookup NOLOGIN;
    END IF;
    -- The migration role must be a member to make auth_lookup the function's
    -- owner (not implied by CREATEROLE since PostgreSQL 16, e.g. on Azure).
    EXECUTE format('GRANT auth_lookup TO %I', current_user);
END
$$;
-- +goose StatementEnd

ALTER ROLE auth_lookup NOLOGIN NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE NOREPLICATION NOINHERIT;

GRANT USAGE ON SCHEMA public TO auth_lookup;
GRANT SELECT (id, tenant_id, email) ON users TO auth_lookup;

-- RLS still applies to auth_lookup (FORCE): this policy lets it read every
-- row, but only through the columns granted above.
CREATE POLICY login_lookup ON users
    FOR SELECT TO auth_lookup
    USING (true);

-- Returns at most 10 candidates (domain.MaxLoginCandidates): each one costs
-- a password verification.
CREATE FUNCTION login_candidates(p_email text)
    RETURNS TABLE (tenant_id uuid, user_id uuid)
    LANGUAGE sql STABLE SECURITY DEFINER
    SET search_path = pg_catalog, pg_temp
    AS $$
        SELECT u.tenant_id, u.id
        FROM public.users u
        WHERE u.email = p_email
        ORDER BY u.tenant_id
        LIMIT 10
    $$;

ALTER FUNCTION login_candidates(text) OWNER TO auth_lookup;
REVOKE ALL ON FUNCTION login_candidates(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION login_candidates(text) TO app_user;

-- +goose Down
DROP FUNCTION login_candidates(text);
DROP POLICY login_lookup ON users;
REVOKE SELECT (id, tenant_id, email) ON users FROM auth_lookup;
REVOKE USAGE ON SCHEMA public FROM auth_lookup;

-- The role is kept if it still has privileges in another database of the cluster.
-- +goose StatementBegin
DO $$
BEGIN
    DROP ROLE IF EXISTS auth_lookup;
EXCEPTION WHEN dependent_objects_still_exist THEN
    RAISE NOTICE 'auth_lookup still has privileges in another database; not dropped';
END
$$;
-- +goose StatementEnd

DROP TABLE refresh_tokens;
