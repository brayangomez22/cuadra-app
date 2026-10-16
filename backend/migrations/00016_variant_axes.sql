-- Catalog: variant axes defined by each tenant (size, color,
-- presentation...) and the values each one allows. Until now a variant could
-- name any axis and any value, so "M" and "m" were two sizes; from now on
-- the domain checks a variant's axis values against its tenant's axes and
-- stores each value as the axis spells it.
--
-- Each existing tenant gets the axes its variants already use, with every
-- value they use. Values that differ only in case become one, with the first
-- spelling in byte order. The variants are not rewritten: one that keeps the
-- other spelling gets the axis's on its next edit.

-- +goose Up
CREATE TABLE variant_axes (
    id             uuid        PRIMARY KEY,
    tenant_id      uuid        NOT NULL REFERENCES tenants (id),
    -- Normalized by the domain: trimmed and lowercase, as the variants'
    -- axis_values name it. Never changed.
    name           text        NOT NULL CHECK (name <> '' AND name = lower(name)),
    -- In their order. The domain keeps them distinct regardless of case.
    allowed_values text[]      NOT NULL CHECK (cardinality(allowed_values) > 0),
    created_at     timestamptz NOT NULL,
    -- The adapter maps a violation of this constraint (by name) to ErrAxisNameTaken.
    CONSTRAINT variant_axes_tenant_id_name_key UNIQUE (tenant_id, name)
);

ALTER TABLE variant_axes ENABLE ROW LEVEL SECURITY;
ALTER TABLE variant_axes FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON variant_axes
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- Backfill. FORCE ROW LEVEL SECURITY applies to the owner too, and the owner
-- running migrations on Azure is not a superuser: with no tenant set, the
-- copy would read zero variants and succeed. FORCE is lifted for the copy
-- and the count check, and restored in the same transaction.
ALTER TABLE product_variants NO FORCE ROW LEVEL SECURITY;
ALTER TABLE variant_axes NO FORCE ROW LEVEL SECURITY;

-- The id is a UUID v7 built as in 00014 (Postgres 16 has no uuidv7()).
INSERT INTO variant_axes (id, tenant_id, name, allowed_values, created_at)
SELECT encode(
           set_bit(set_bit(
               overlay(uuid_send(gen_random_uuid())
                       PLACING substring(int8send((extract(epoch FROM clock_timestamp()) * 1000)::bigint) FROM 3)
                       FROM 1 FOR 6),
               52, 1), 53, 1),
           'hex')::uuid,
       used.tenant_id, used.name, array_agg(used.value ORDER BY used.value COLLATE "C"), now()
FROM (
    SELECT v.tenant_id, axis.key AS name, min(axis.value COLLATE "C") AS value
    FROM product_variants v
    CROSS JOIN LATERAL jsonb_each_text(v.axis_values) AS axis
    GROUP BY v.tenant_id, axis.key, lower(axis.value)
) AS used
GROUP BY used.tenant_id, used.name;

-- Checked before FORCE is back: with it, every count would read zero.
-- +goose StatementBegin
DO $$
DECLARE
    used_axes   bigint;
    stored_axes bigint;
    uncovered   bigint;
    too_long    bigint;
BEGIN
    SELECT count(*) INTO used_axes
    FROM (SELECT DISTINCT v.tenant_id, axis.key
          FROM product_variants v CROSS JOIN LATERAL jsonb_object_keys(v.axis_values) AS axis (key)) AS used;
    SELECT count(*) INTO stored_axes FROM variant_axes;
    IF stored_axes <> used_axes THEN
        RAISE EXCEPTION 'variant_axes backfill: the variants use % axes but % were stored', used_axes, stored_axes;
    END IF;
    SELECT count(*) INTO uncovered
    FROM product_variants v
    CROSS JOIN LATERAL jsonb_each_text(v.axis_values) AS axis
    WHERE NOT EXISTS (
        SELECT 1 FROM variant_axes a, unnest(a.allowed_values) AS allowed (value)
        WHERE a.tenant_id = v.tenant_id AND a.name = axis.key AND lower(allowed.value) = lower(axis.value)
    );
    IF uncovered <> 0 THEN
        RAISE EXCEPTION 'variant_axes backfill: % variant axis values are not allowed by their axis', uncovered;
    END IF;
    -- The domain allows up to 200 values per axis.
    SELECT count(*) INTO too_long FROM variant_axes WHERE cardinality(allowed_values) > 200;
    IF too_long <> 0 THEN
        RAISE EXCEPTION 'variant_axes backfill: % axes have more than 200 values', too_long;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE product_variants FORCE ROW LEVEL SECURITY;
ALTER TABLE variant_axes FORCE ROW LEVEL SECURITY;

-- +goose Down
DROP TABLE variant_axes;
