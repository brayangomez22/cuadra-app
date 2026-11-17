-- Comments the users send from the app ("Danos tu opinión", T40), with the
-- screen and the app version they were on. Per tenant, with RLS: the export
-- and the deletion of a tenant (T49) take them along. notified_at is set
-- when the team was told (the feedback.notify job).

-- +goose Up
CREATE TABLE feedback (
    id          uuid        PRIMARY KEY,
    tenant_id   uuid        NOT NULL REFERENCES tenants (id),
    user_id     uuid        NOT NULL REFERENCES users (id),
    message     text        NOT NULL CHECK (char_length(message) BETWEEN 1 AND 2000),
    screen      text        NOT NULL CHECK (char_length(screen) BETWEEN 1 AND 200),
    app_version text        NOT NULL CHECK (char_length(app_version) BETWEEN 1 AND 50),
    created_at  timestamptz NOT NULL,
    notified_at timestamptz
);

CREATE INDEX feedback_tenant_created_idx ON feedback (tenant_id, created_at DESC);

ALTER TABLE feedback ENABLE ROW LEVEL SECURITY;
ALTER TABLE feedback FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON feedback
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- +goose Down
DROP TABLE feedback;
