-- Audit log (T38): who voided, discounted, changed a price, adjusted stock,
-- closed the cash with a difference or changed the configuration. Each
-- event is written in the transaction of the operation it records, and is
-- immutable: app_user can only insert and read (the demo purge deletes as
-- demo_purge, like the kardex). Isolated per tenant with Row-Level
-- Security.
--
-- The action is checked against the closed catalog by the domain
-- (docs/auditoria.md); details holds the figure the business reports add
-- up, every value a string (amounts as decimal strings, never floats).
-- The indexes let the user management screen (T41) filter by action or
-- user and the reports (T42) add up a period without reading the table.

-- +goose Up
CREATE TABLE audit_events (
    id          uuid        PRIMARY KEY,
    tenant_id   uuid        NOT NULL REFERENCES tenants (id),
    user_id     uuid        NOT NULL,
    role        text        NOT NULL CHECK (role <> ''),
    action      text        NOT NULL CHECK (action ~ '^[a-z_]+\.[a-z_]+$'),
    entity_type text        NOT NULL CHECK (entity_type <> ''),
    entity_id   uuid,
    location_id uuid,
    occurred_at timestamptz NOT NULL,
    -- No foreign key to users, like the sales' and the kardex's users: the
    -- event keeps who did it even if the user is gone.
    details     jsonb       NOT NULL CHECK (jsonb_typeof(details) = 'object')
);

CREATE INDEX audit_events_tenant_action_idx ON audit_events (tenant_id, action, occurred_at);
CREATE INDEX audit_events_tenant_user_idx ON audit_events (tenant_id, user_id, occurred_at);
CREATE INDEX audit_events_tenant_occurred_idx ON audit_events (tenant_id, occurred_at DESC, id DESC);

ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON audit_events
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

REVOKE UPDATE, DELETE ON audit_events FROM app_user;

-- +goose Down
DROP TABLE audit_events;
