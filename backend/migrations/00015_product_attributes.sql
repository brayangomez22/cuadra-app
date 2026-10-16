-- Catalog: product attributes defined by each tenant (brand, material,
-- active ingredient...). A product keeps its values in products.attributes,
-- keyed by the definition's key; the domain checks them against the
-- tenant's definitions when they are written. No column is ever added for
-- the attributes of one kind of business.

-- +goose Up
CREATE TABLE attribute_definitions (
    id         uuid        PRIMARY KEY,
    tenant_id  uuid        NOT NULL REFERENCES tenants (id),
    -- A lowercase slug, never changed: products keep their values under it.
    key        text        NOT NULL CHECK (key ~ '^[a-z][a-z0-9_]*$'),
    label      text        NOT NULL CHECK (label <> ''),
    -- Never changed, like the key.
    type       text        NOT NULL CHECK (type IN ('text', 'number', 'list', 'boolean')),
    -- The options of a list attribute, in their order; empty for the others.
    options    text[]      NOT NULL DEFAULT '{}',
    required   boolean     NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT attribute_definitions_options_check CHECK ((type = 'list') = (cardinality(options) > 0)),
    -- The adapter maps a violation of this constraint (by name) to ErrAttributeKeyTaken.
    CONSTRAINT attribute_definitions_tenant_id_key_key UNIQUE (tenant_id, key)
);

ALTER TABLE attribute_definitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE attribute_definitions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON attribute_definitions
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- Existing products start without attributes. A constant default does not
-- rewrite the table, so no backfill under RLS is needed.
ALTER TABLE products
    ADD COLUMN attributes jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(attributes) = 'object');

-- Serves the "attribute in use" checks and, later, the filters by attribute.
CREATE INDEX products_attributes_idx ON products USING gin (attributes);

-- +goose Down
DROP INDEX products_attributes_idx;
ALTER TABLE products DROP COLUMN attributes;
DROP TABLE attribute_definitions;
