-- Catalog: alternate units of a variant (a box of 12, a roll of 100 m, the
-- gram of something counted in kilograms), each with how many base units it
-- holds, its own price and an optional barcode. The stock stays in the
-- product's base unit; sales and receipts convert. The tables belong to the
-- core catalog; the unit_conversions module only lets a tenant manage them.
--
-- A sale line keeps the unit it was sold in: its code and its factor, frozen
-- like the product's name, so the receipt of an old sale does not change if
-- the unit does. NULL unit_code is the product's base unit, as every line
-- sold until now.
--
-- Existing tenants get the unit_conversions module if their business
-- template has it (ferreteria, tienda and generica): the template only
-- applies its modules at sign-up.

-- +goose Up
CREATE TABLE variant_units (
    tenant_id  uuid          NOT NULL REFERENCES tenants (id),
    variant_id uuid          NOT NULL,
    unit_id    uuid          NOT NULL,
    -- How many base units one alternate unit holds.
    factor     numeric(18,4) NOT NULL CHECK (factor > 0),
    -- Price of one alternate unit, without IVA.
    price      numeric(18,4) NOT NULL CHECK (price > 0),
    barcode    text          CHECK (barcode <> ''),
    PRIMARY KEY (tenant_id, variant_id, unit_id),
    CONSTRAINT variant_units_variant_fkey FOREIGN KEY (tenant_id, variant_id)
        REFERENCES product_variants (tenant_id, id),
    -- Deleting a unit in use as an alternate unit is ErrUnitInUse.
    CONSTRAINT variant_units_unit_fkey FOREIGN KEY (tenant_id, unit_id)
        REFERENCES units_of_measure (tenant_id, id),
    -- The adapter maps a violation of this constraint (by name) to
    -- ErrBarcodeTaken. A barcode must not repeat a variant's either: the
    -- repository checks it, since a constraint cannot span both tables.
    CONSTRAINT variant_units_barcode_key UNIQUE (tenant_id, barcode)
);

ALTER TABLE variant_units ENABLE ROW LEVEL SECURITY;
ALTER TABLE variant_units FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON variant_units
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

ALTER TABLE sale_lines
    ADD COLUMN unit_code   text          CHECK (unit_code <> ''),
    ADD COLUMN unit_factor numeric(18,4) NOT NULL DEFAULT 1 CHECK (unit_factor > 0),
    ADD CONSTRAINT sale_lines_base_unit_factor_check CHECK (unit_code IS NOT NULL OR unit_factor = 1);

-- Backfill. FORCE ROW LEVEL SECURITY applies to the owner too, and the owner
-- running migrations on Azure is not a superuser: with no tenant set, the
-- copy would read zero tenants and succeed. FORCE is lifted for the copy
-- and the count check, and restored in the same transaction.
ALTER TABLE tenants NO FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_modules NO FORCE ROW LEVEL SECURITY;

INSERT INTO tenant_modules (tenant_id, module_id, enabled_at, enabled_by)
SELECT id, 'unit_conversions', now(), NULL
FROM tenants
WHERE business_template IN ('ferreteria', 'tienda', 'generica')
ON CONFLICT DO NOTHING;

-- Checked before FORCE is back: with it, every count would read zero.
-- +goose StatementBegin
DO $$
DECLARE
    expected bigint;
    enabled  bigint;
BEGIN
    SELECT count(*) INTO expected FROM tenants WHERE business_template IN ('ferreteria', 'tienda', 'generica');
    SELECT count(*) INTO enabled
    FROM tenant_modules m
    JOIN tenants t ON t.id = m.tenant_id
    WHERE m.module_id = 'unit_conversions' AND t.business_template IN ('ferreteria', 'tienda', 'generica');
    IF enabled <> expected THEN
        RAISE EXCEPTION 'unit_conversions backfill: % tenants should have it but % do', expected, enabled;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_modules FORCE ROW LEVEL SECURITY;

-- +goose Down
-- Fails if a sale was made in an alternate unit: its quantity is not in the
-- base unit, and dropping the unit would change what the line says.
-- +goose StatementBegin
DO $$
BEGIN
    ALTER TABLE sale_lines NO FORCE ROW LEVEL SECURITY;
    IF EXISTS (SELECT 1 FROM sale_lines WHERE unit_code IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot roll back 00020: there are sale lines in an alternate unit';
    END IF;
    ALTER TABLE sale_lines FORCE ROW LEVEL SECURITY;
END
$$;
-- +goose StatementEnd

ALTER TABLE tenant_modules NO FORCE ROW LEVEL SECURITY;
DELETE FROM tenant_modules WHERE module_id = 'unit_conversions';
ALTER TABLE tenant_modules FORCE ROW LEVEL SECURITY;

ALTER TABLE sale_lines
    DROP CONSTRAINT sale_lines_base_unit_factor_check,
    DROP COLUMN unit_factor,
    DROP COLUMN unit_code;

DROP TABLE variant_units;
