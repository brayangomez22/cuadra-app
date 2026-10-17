-- Business templates: the template (backend/templates/business/*.yaml) each
-- tenant signed up with. The templates live in code, so the column has no
-- foreign key; the sign-up checks that the template exists.
--
-- Every existing tenant gets the ferreteria template, the pilot's first
-- market. Its modules are not applied here: ferreteria declares no module
-- today, and the tenants keep the variants module 00017 gave them because
-- they could already add variants.

-- +goose Up
ALTER TABLE tenants ADD COLUMN business_template text
    CHECK (business_template ~ '^[a-z][a-z0-9_]{0,63}$');

-- Backfill. FORCE ROW LEVEL SECURITY applies to the owner too, and the owner
-- running migrations on Azure is not a superuser: with no tenant set, the
-- update would touch zero tenants and succeed. FORCE is lifted for the update
-- and the count check, and restored in the same transaction.
ALTER TABLE tenants NO FORCE ROW LEVEL SECURITY;

UPDATE tenants SET business_template = 'ferreteria';

-- Checked before FORCE is back: with it, every count would read zero.
-- +goose StatementBegin
DO $$
DECLARE
    tenant_count  bigint;
    with_template bigint;
BEGIN
    SELECT count(*) INTO tenant_count FROM tenants;
    SELECT count(*) INTO with_template FROM tenants WHERE business_template = 'ferreteria';
    IF with_template <> tenant_count THEN
        RAISE EXCEPTION 'tenants backfill: % tenants but % have a business template', tenant_count, with_template;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE tenants ALTER COLUMN business_template SET NOT NULL;

ALTER TABLE tenants FORCE ROW LEVEL SECURITY;

-- +goose Down
ALTER TABLE tenants DROP COLUMN business_template;
