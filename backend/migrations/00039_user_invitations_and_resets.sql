-- Links to join a tenant and to choose a new password (T41). Only the
-- SHA-256 of each link's secret is stored; the tenant travels in the link,
-- so it is looked up within its tenant (RLS) before anyone is
-- authenticated, like the refresh tokens.

-- +goose Up
CREATE TABLE user_invitations (
    id          uuid        PRIMARY KEY,
    tenant_id   uuid        NOT NULL REFERENCES tenants (id),
    email       text        NOT NULL,
    name        text        NOT NULL,
    role        text        NOT NULL CHECK (role IN ('owner', 'admin', 'cashier', 'warehouse')),
    invited_by  uuid        NOT NULL REFERENCES users (id),
    token_hash  bytea       NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    expires_at  timestamptz NOT NULL,
    created_at  timestamptz NOT NULL,
    accepted_at timestamptz,
    revoked_at  timestamptz,
    CHECK (accepted_at IS NULL OR revoked_at IS NULL)
);

CREATE INDEX user_invitations_open_idx ON user_invitations (tenant_id, created_at DESC)
    WHERE accepted_at IS NULL AND revoked_at IS NULL;

ALTER TABLE user_invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_invitations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON user_invitations
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE TABLE password_resets (
    id         uuid        PRIMARY KEY,
    tenant_id  uuid        NOT NULL REFERENCES tenants (id),
    user_id    uuid        NOT NULL REFERENCES users (id),
    issued_by  uuid        NOT NULL REFERENCES users (id),
    token_hash bytea       NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    used_at    timestamptz
);

CREATE INDEX password_resets_unused_idx ON password_resets (tenant_id, user_id) WHERE used_at IS NULL;

ALTER TABLE password_resets ENABLE ROW LEVEL SECURITY;
ALTER TABLE password_resets FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON password_resets
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- +goose Down
DROP TABLE password_resets;
DROP TABLE user_invitations;
