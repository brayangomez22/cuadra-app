-- Catalog: SKU, barcode, cost and price now live only in product_variants
-- (00010). Dropping the columns also drops their constraints
-- (products_tenant_id_sku_key, products_tenant_id_barcode_key and the checks).

-- +goose Up
ALTER TABLE products
    DROP COLUMN sku,
    DROP COLUMN barcode,
    DROP COLUMN cost,
    DROP COLUMN price;

-- +goose Down
-- Restores the columns from each product's default variant, lifting FORCE
-- ROW LEVEL SECURITY for the copy as 00010 does.
ALTER TABLE products
    ADD COLUMN sku     text,
    ADD COLUMN barcode text,
    ADD COLUMN cost    numeric(18,4),
    ADD COLUMN price   numeric(18,4);

ALTER TABLE products NO FORCE ROW LEVEL SECURITY;
ALTER TABLE product_variants NO FORCE ROW LEVEL SECURITY;

UPDATE products p
SET sku = v.sku, barcode = v.barcode, cost = v.cost, price = v.price
FROM product_variants v
WHERE v.tenant_id = p.tenant_id AND v.product_id = p.id AND v.is_default;

-- +goose StatementBegin
DO $$
DECLARE
    missing bigint;
BEGIN
    SELECT count(*) INTO missing FROM products WHERE sku IS NULL OR cost IS NULL OR price IS NULL;
    IF missing <> 0 THEN
        RAISE EXCEPTION 'products restore: % products without a default variant to copy from', missing;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE products FORCE ROW LEVEL SECURITY;
ALTER TABLE product_variants FORCE ROW LEVEL SECURITY;

ALTER TABLE products
    ALTER COLUMN sku SET NOT NULL,
    ALTER COLUMN cost SET NOT NULL,
    ALTER COLUMN price SET NOT NULL,
    ADD CONSTRAINT products_sku_check CHECK (sku = upper(sku)),
    ADD CONSTRAINT products_barcode_check CHECK (barcode <> ''),
    ADD CONSTRAINT products_cost_check CHECK (cost >= 0),
    ADD CONSTRAINT products_price_check CHECK (price > 0),
    ADD CONSTRAINT products_tenant_id_sku_key UNIQUE (tenant_id, sku);

CREATE UNIQUE INDEX products_tenant_id_barcode_key ON products (tenant_id, barcode) WHERE barcode IS NOT NULL;
