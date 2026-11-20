-- Who hid their getting-started checklist (T44), and which steps were
-- pending at that moment: a critical step that goes missing afterwards (a
-- numbering resolution that expires) brings the checklist back. One row per
-- user, replaced each time they hide it again. The checklist itself is not
-- stored: it is computed from the tenant's data. No data moves.

-- +goose Up
CREATE TABLE onboarding_dismissals (
    tenant_id     uuid        NOT NULL REFERENCES tenants (id),
    user_id       uuid        NOT NULL REFERENCES users (id),
    dismissed_at  timestamptz NOT NULL,
    pending_steps text[]      NOT NULL,
    PRIMARY KEY (tenant_id, user_id)
);

ALTER TABLE onboarding_dismissals ENABLE ROW LEVEL SECURITY;
ALTER TABLE onboarding_dismissals FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON onboarding_dismissals
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- +goose Down
DROP TABLE onboarding_dismissals;
