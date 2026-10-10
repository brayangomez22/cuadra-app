-- Catalog: categories and products, isolated per tenant with Row-Level Security.
--
-- Foreign keys between business tables include tenant_id: Postgres checks
-- foreign keys without Row-Level Security, so a plain category_id reference
-- would let a product point to another tenant's category.

-- +goose Up
CREATE EXTENSION IF NOT EXISTS unaccent;

-- unaccent() is only STABLE (it reads its dictionary through the search
-- path); this wrapper names the dictionary explicitly so it can be IMMUTABLE
-- and back a generated column and its index.
CREATE FUNCTION f_unaccent(text) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
    AS $$ SELECT public.unaccent('public.unaccent'::regdictionary, $1) $$;

CREATE TABLE categories (
    id         uuid        PRIMARY KEY,
    tenant_id  uuid        NOT NULL REFERENCES tenants (id),
    name       text        NOT NULL,
    parent_id  uuid,
    created_at timestamptz NOT NULL,
    -- Target of the tenant-scoped foreign keys below.
    CONSTRAINT categories_tenant_id_id_key UNIQUE (tenant_id, id),
    -- The adapter maps a violation of this constraint (by name) to ErrCategoryNotFound.
    CONSTRAINT categories_parent_fkey FOREIGN KEY (tenant_id, parent_id) REFERENCES categories (tenant_id, id),
    CONSTRAINT categories_not_own_parent CHECK (parent_id <> id)
);

CREATE INDEX categories_tenant_id_parent_id_idx ON categories (tenant_id, parent_id);

ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE categories FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON categories
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE TABLE products (
    id          uuid          PRIMARY KEY,
    tenant_id   uuid          NOT NULL REFERENCES tenants (id),
    -- Normalized by the domain (uppercase, no spaces).
    sku         text          NOT NULL CHECK (sku = upper(sku)),
    -- NULL when the product has none, so many products can lack one.
    barcode     text          CHECK (barcode <> ''),
    name        text          NOT NULL,
    -- Lowercase and without accents, so "presion" finds "presión". It has no
    -- trigram index: under Row-Level Security Postgres cannot use one for
    -- LIKE (not LEAKPROOF), so a search scans the tenant's rows (docs/decisiones.md).
    search_name text          NOT NULL GENERATED ALWAYS AS (public.f_unaccent(lower(name))) STORED,
    description text          NOT NULL,
    category_id uuid,
    base_unit   text          NOT NULL CHECK (base_unit IN ('und', 'm', 'kg', 'l', 'caja', 'rollo', 'bulto', 'galon')),
    cost        numeric(18,4) NOT NULL CHECK (cost >= 0),
    -- Before VAT (the taxable base).
    price       numeric(18,4) NOT NULL CHECK (price > 0),
    tax_rate    numeric(5,4)  NOT NULL CHECK (tax_rate >= 0 AND tax_rate < 1),
    active      boolean       NOT NULL,
    created_at  timestamptz   NOT NULL,
    -- The adapter maps violations of these constraints (by name) to domain errors.
    CONSTRAINT products_tenant_id_sku_key UNIQUE (tenant_id, sku),
    CONSTRAINT products_category_fkey FOREIGN KEY (tenant_id, category_id) REFERENCES categories (tenant_id, id)
);

CREATE UNIQUE INDEX products_tenant_id_barcode_key ON products (tenant_id, barcode) WHERE barcode IS NOT NULL;
CREATE INDEX products_tenant_id_category_id_idx ON products (tenant_id, category_id);

ALTER TABLE products ENABLE ROW LEVEL SECURITY;
ALTER TABLE products FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON products
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- +goose Down
DROP TABLE products;
DROP TABLE categories;
DROP FUNCTION f_unaccent(text);
DROP EXTENSION IF EXISTS unaccent;
