-- Catalog: product variants. A product groups one or more variants, and the
-- variant is what is sold and counted: it holds the SKU, barcode, cost and
-- price. Each existing product gets a default variant with its data.
--
-- The default variant of an existing product reuses the product's UUID, so
-- the product ids already stored in stock, the kardex and sale lines are
-- valid variant ids and those immutable tables are not rewritten. New
-- products create their default variant with its own UUID v7: no code may
-- assume variant.id = product.id (docs/decisiones.md).
--
-- products.sku, barcode, cost and price stay until 00011, so this step can
-- be rolled back on its own.

-- +goose Up
CREATE TABLE product_variants (
    id          uuid          PRIMARY KEY,
    tenant_id   uuid          NOT NULL REFERENCES tenants (id),
    product_id  uuid          NOT NULL,
    -- Normalized by the domain (uppercase, no spaces).
    sku         text          NOT NULL CHECK (sku = upper(sku)),
    -- NULL when the variant has none, so many variants can lack one.
    barcode     text          CHECK (barcode <> ''),
    cost        numeric(18,4) NOT NULL CHECK (cost >= 0),
    -- Before VAT (the taxable base).
    price       numeric(18,4) NOT NULL CHECK (price > 0),
    -- The variant's value per axis ({"color": "Azul"}), normalized by the
    -- domain; {} for the default variant of a simple product.
    axis_values jsonb         NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(axis_values) = 'object'),
    is_default  boolean       NOT NULL,
    active      boolean       NOT NULL,
    created_at  timestamptz   NOT NULL,
    -- Target of the tenant-scoped foreign keys of stock and sales (T24b, T24c).
    CONSTRAINT product_variants_tenant_id_id_key UNIQUE (tenant_id, id),
    -- The adapter maps violations of these constraints (by name) to domain errors.
    CONSTRAINT product_variants_tenant_id_sku_key UNIQUE (tenant_id, sku),
    CONSTRAINT product_variants_axis_values_key UNIQUE (tenant_id, product_id, axis_values),
    CONSTRAINT product_variants_product_fkey FOREIGN KEY (tenant_id, product_id) REFERENCES products (tenant_id, id)
);

CREATE UNIQUE INDEX product_variants_tenant_id_barcode_key ON product_variants (tenant_id, barcode) WHERE barcode IS NOT NULL;
-- At most one default variant per product; the domain demands exactly one.
CREATE UNIQUE INDEX product_variants_default_key ON product_variants (tenant_id, product_id) WHERE is_default;

ALTER TABLE product_variants ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_variants FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON product_variants
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- Backfill. FORCE ROW LEVEL SECURITY applies to the owner too, and the owner
-- running migrations on Azure is not a superuser: with no tenant set, the
-- copy would see zero rows and succeed. FORCE is lifted for the copy and the
-- count check, and restored in the same transaction.
ALTER TABLE products NO FORCE ROW LEVEL SECURITY;
ALTER TABLE product_variants NO FORCE ROW LEVEL SECURITY;

INSERT INTO product_variants (id, tenant_id, product_id, sku, barcode, cost, price, axis_values, is_default, active, created_at)
SELECT id, tenant_id, id, sku, barcode, cost, price, '{}', true, true, created_at
FROM products;

-- Checked before FORCE is back: with it, both counts would read zero.
-- +goose StatementBegin
DO $$
DECLARE
    product_count bigint;
    variant_count bigint;
BEGIN
    SELECT count(*) INTO product_count FROM products;
    SELECT count(*) INTO variant_count
    FROM product_variants v JOIN products p ON p.id = v.id AND p.tenant_id = v.tenant_id
    WHERE v.is_default AND v.product_id = p.id
      AND v.sku = p.sku AND v.barcode IS NOT DISTINCT FROM p.barcode
      AND v.cost = p.cost AND v.price = p.price;
    IF product_count <> variant_count THEN
        RAISE EXCEPTION 'product_variants backfill: % products but % matching default variants', product_count, variant_count;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE products FORCE ROW LEVEL SECURITY;
ALTER TABLE product_variants FORCE ROW LEVEL SECURITY;

-- +goose Down
DROP TABLE product_variants;
