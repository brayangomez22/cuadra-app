-- The owner's consent to share the app's usage analytics with Cuadra (T40,
-- Ley 1581): off for every tenant, existing or new, until the owner turns it
-- on in the settings. It adds a column with a constant default and moves no
-- data.

-- +goose Up
ALTER TABLE tenants ADD COLUMN analytics_consent boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE tenants DROP COLUMN analytics_consent;
