-- Catalog: units of measure per tenant. They replace the fixed list that
-- products.base_unit was checked against: each existing tenant gets the
-- units of that list, and each product points to its tenant's unit with the
-- same code. New tenants get the same units at sign-up (T24d) until the
-- business templates seed theirs (T24g).

-- +goose Up
CREATE TABLE units_of_measure (
    id         uuid        PRIMARY KEY,
    tenant_id  uuid        NOT NULL REFERENCES tenants (id),
    -- Normalized by the domain: lowercase, no spaces. Shown next to a
    -- quantity ("2,5 m").
    code       text        NOT NULL CHECK (code <> '' AND code = lower(code)),
    name       text        NOT NULL CHECK (name <> ''),
    created_at timestamptz NOT NULL,
    -- Target of the tenant-scoped foreign key of products.
    CONSTRAINT units_of_measure_tenant_id_id_key UNIQUE (tenant_id, id),
    -- The adapter maps a violation of this constraint (by name) to ErrUnitCodeTaken.
    CONSTRAINT units_of_measure_tenant_id_code_key UNIQUE (tenant_id, code)
);

ALTER TABLE units_of_measure ENABLE ROW LEVEL SECURITY;
ALTER TABLE units_of_measure FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON units_of_measure
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

ALTER TABLE products ADD COLUMN base_unit_id uuid;

-- Backfill. FORCE ROW LEVEL SECURITY applies to the owner too, and the owner
-- running migrations on Azure is not a superuser: with no tenant set, the
-- seed would read zero tenants and succeed. FORCE is lifted for the seed,
-- the update and the count check, and restored in the same transaction.
ALTER TABLE tenants NO FORCE ROW LEVEL SECURITY;
ALTER TABLE products NO FORCE ROW LEVEL SECURITY;
ALTER TABLE units_of_measure NO FORCE ROW LEVEL SECURITY;

-- Postgres 16 has no uuidv7(): the id is a random UUID with the current
-- Unix time in milliseconds in its first 48 bits and the version bits set
-- to 7 (RFC 9562).
INSERT INTO units_of_measure (id, tenant_id, code, name, created_at)
SELECT encode(
           set_bit(set_bit(
               overlay(uuid_send(gen_random_uuid())
                       PLACING substring(int8send((extract(epoch FROM clock_timestamp()) * 1000)::bigint) FROM 3)
                       FROM 1 FOR 6),
               52, 1), 53, 1),
           'hex')::uuid,
       t.id, u.code, u.name, now()
FROM tenants t
CROSS JOIN (VALUES
    ('und', 'Unidad'), ('m', 'Metro'), ('kg', 'Kilogramo'), ('l', 'Litro'),
    ('caja', 'Caja'), ('rollo', 'Rollo'), ('bulto', 'Bulto'), ('galon', 'Galón')
) AS u (code, name);

UPDATE products p
SET base_unit_id = u.id
FROM units_of_measure u
WHERE u.tenant_id = p.tenant_id AND u.code = p.base_unit;

-- Checked before FORCE is back: with it, every count would read zero.
-- +goose StatementBegin
DO $$
DECLARE
    tenant_count     bigint;
    unit_count       bigint;
    product_count    bigint;
    unassigned_count bigint;
BEGIN
    SELECT count(*) INTO tenant_count FROM tenants;
    SELECT count(*) INTO unit_count FROM units_of_measure;
    IF unit_count <> tenant_count * 8 THEN
        RAISE EXCEPTION 'units_of_measure backfill: % tenants need % units but % were stored',
            tenant_count, tenant_count * 8, unit_count;
    END IF;
    SELECT count(*) INTO product_count FROM products;
    SELECT count(*) INTO unassigned_count
    FROM products p
    WHERE NOT EXISTS (
        SELECT 1 FROM units_of_measure u
        WHERE u.id = p.base_unit_id AND u.tenant_id = p.tenant_id AND u.code = p.base_unit
    );
    IF unassigned_count <> 0 THEN
        RAISE EXCEPTION 'units_of_measure backfill: % of % products without the unit of their code',
            unassigned_count, product_count;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
ALTER TABLE products FORCE ROW LEVEL SECURITY;
ALTER TABLE units_of_measure FORCE ROW LEVEL SECURITY;

ALTER TABLE products
    ALTER COLUMN base_unit_id SET NOT NULL,
    -- The adapter maps a violation of this constraint (by name) to
    -- ErrUnitNotFound, and any foreign key violation on a unit's delete to
    -- ErrUnitInUse.
    ADD CONSTRAINT products_base_unit_fkey FOREIGN KEY (tenant_id, base_unit_id) REFERENCES units_of_measure (tenant_id, id),
    DROP COLUMN base_unit;

CREATE INDEX products_tenant_id_base_unit_id_idx ON products (tenant_id, base_unit_id);

-- +goose Down
-- Restores base_unit from the code of each product's unit, lifting FORCE
-- ROW LEVEL SECURITY for the copy as the Up does. It fails if a product uses
-- a unit outside the fixed list, which base_unit cannot hold.
ALTER TABLE products ADD COLUMN base_unit text;

ALTER TABLE products NO FORCE ROW LEVEL SECURITY;
ALTER TABLE units_of_measure NO FORCE ROW LEVEL SECURITY;

UPDATE products p
SET base_unit = u.code
FROM units_of_measure u
WHERE u.tenant_id = p.tenant_id AND u.id = p.base_unit_id;

-- +goose StatementBegin
DO $$
DECLARE
    outside bigint;
BEGIN
    SELECT count(*) INTO outside FROM products
    WHERE base_unit IS NULL OR base_unit NOT IN ('und', 'm', 'kg', 'l', 'caja', 'rollo', 'bulto', 'galon');
    IF outside <> 0 THEN
        RAISE EXCEPTION 'units_of_measure restore: % products use a unit outside the fixed list', outside;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE products FORCE ROW LEVEL SECURITY;
ALTER TABLE units_of_measure FORCE ROW LEVEL SECURITY;

ALTER TABLE products
    ALTER COLUMN base_unit SET NOT NULL,
    ADD CONSTRAINT products_base_unit_check CHECK (base_unit IN ('und', 'm', 'kg', 'l', 'caja', 'rollo', 'bulto', 'galon')),
    DROP COLUMN base_unit_id;

DROP TABLE units_of_measure;
