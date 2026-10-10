-- Identity: tenants and users, isolated per tenant with Row-Level Security.
--
-- Every business table follows the same pattern: ENABLE + FORCE ROW LEVEL
-- SECURITY and a policy comparing its tenant column with current_tenant_id(),
-- both to read (USING) and to write (WITH CHECK).

-- +goose Up

-- The tenant of the current transaction, set by db.WithTenantTx with
-- set_config('app.tenant_id', ..., true). Fail-closed: when it is not set (or
-- left empty by a finished transaction on the same session) it returns NULL,
-- which matches no row, instead of raising an error.
CREATE FUNCTION current_tenant_id() RETURNS uuid
    LANGUAGE sql STABLE
    AS $$ SELECT NULLIF(current_setting('app.tenant_id', true), '')::uuid $$;

CREATE TABLE tenants (
    id         uuid        PRIMARY KEY,
    name       text        NOT NULL,
    -- Canonical "number-DV". Not unique: a unique index would tell one tenant
    -- that another already registered the NIT, and a business may have two accounts.
    nit        text        NOT NULL,
    status     text        NOT NULL CHECK (status IN ('active', 'suspended')),
    created_at timestamptz NOT NULL
);

ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;

-- A tenant sees and writes only its own row. Creating one (sign-up) runs in a
-- transaction already scoped to the new tenant's id.
CREATE POLICY tenant_isolation ON tenants
    USING (id = current_tenant_id())
    WITH CHECK (id = current_tenant_id());

CREATE TABLE users (
    id            uuid        PRIMARY KEY,
    tenant_id     uuid        NOT NULL REFERENCES tenants (id),
    -- Normalized by the domain (trimmed, lowercase); the check keeps the
    -- unique index case-insensitive.
    email         text        NOT NULL CHECK (email = lower(email)),
    name          text        NOT NULL,
    password_hash text        NOT NULL,
    role          text        NOT NULL CHECK (role IN ('owner', 'admin', 'cashier', 'warehouse')),
    active        boolean     NOT NULL,
    created_at    timestamptz NOT NULL,
    -- The adapter maps a violation of this constraint (by name) to ErrEmailTaken.
    CONSTRAINT users_tenant_id_email_key UNIQUE (tenant_id, email)
);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

-- WITH CHECK also stops an UPDATE from moving a user to another tenant.
CREATE POLICY tenant_isolation ON users
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- +goose Down
DROP TABLE users;
DROP TABLE tenants;
DROP FUNCTION current_tenant_id();
