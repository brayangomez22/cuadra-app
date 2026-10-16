-- Inventory: stock levels and the kardex move from products to variants, and
-- gain an optional lot (lot_id), with no lot logic yet: the lots_expiry
-- module adds it. The core works on the stock without a lot (lot_id NULL).
--
-- product_id becomes variant_id. The default variant of a product from
-- before 00010 has the product's id, so its rows keep their value. A product
-- created after 00010 has a default variant with its own id: its rows move to
-- that variant, the only way the new foreign key holds. The owner may update
-- stock_movements (app_user may not), and the move does not change what any
-- movement says: the default variant is the product as it was sold.
--
-- A level is unique per (tenant, variant, location, lot), with NULLS NOT
-- DISTINCT so two levels without a lot cannot coexist. The table has no
-- primary key: nothing references a level, and lot_id cannot be part of one.

-- +goose Up

-- FORCE ROW LEVEL SECURITY applies to the owner too (see 00010): it is lifted
-- for the move and the checks, and restored in the same transaction.
ALTER TABLE stock_levels NO FORCE ROW LEVEL SECURITY;
ALTER TABLE stock_movements NO FORCE ROW LEVEL SECURITY;
ALTER TABLE product_variants NO FORCE ROW LEVEL SECURITY;

-- The stock and the kardex of each product, to compare after the move.
CREATE TEMP TABLE stock_before ON COMMIT DROP AS
SELECT tenant_id, product_id, sum(quantity) AS quantity, count(*) AS levels
FROM stock_levels GROUP BY tenant_id, product_id;

CREATE TEMP TABLE movements_before ON COMMIT DROP AS
SELECT tenant_id, product_id, sum(quantity) AS quantity, count(*) AS movements
FROM stock_movements GROUP BY tenant_id, product_id;

ALTER TABLE stock_levels DROP CONSTRAINT stock_levels_product_fkey;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_product_fkey;
ALTER TABLE stock_levels RENAME COLUMN product_id TO variant_id;
ALTER TABLE stock_movements RENAME COLUMN product_id TO variant_id;

-- Rows of products created after 00010 move to their default variant.
UPDATE stock_levels l
SET variant_id = v.id
FROM product_variants v
WHERE v.tenant_id = l.tenant_id AND v.product_id = l.variant_id AND v.is_default AND v.id <> l.variant_id;

UPDATE stock_movements m
SET variant_id = v.id
FROM product_variants v
WHERE v.tenant_id = m.tenant_id AND v.product_id = m.variant_id AND v.is_default AND v.id <> m.variant_id;

-- Checked before FORCE is back: with it, every count would read zero.
-- +goose StatementBegin
DO $$
DECLARE
    stock_changed bigint;
    kardex_changed bigint;
BEGIN
    SELECT count(*) INTO stock_changed
    FROM stock_before b
    FULL JOIN (
        SELECT l.tenant_id, v.product_id, sum(l.quantity) AS quantity, count(*) AS levels
        FROM stock_levels l JOIN product_variants v ON v.tenant_id = l.tenant_id AND v.id = l.variant_id
        GROUP BY l.tenant_id, v.product_id
    ) a ON a.tenant_id = b.tenant_id AND a.product_id = b.product_id
    WHERE a.quantity IS DISTINCT FROM b.quantity OR a.levels IS DISTINCT FROM b.levels;
    IF stock_changed <> 0 THEN
        RAISE EXCEPTION 'stock by variant: % products whose stock changed', stock_changed;
    END IF;

    SELECT count(*) INTO kardex_changed
    FROM movements_before b
    FULL JOIN (
        SELECT m.tenant_id, v.product_id, sum(m.quantity) AS quantity, count(*) AS movements
        FROM stock_movements m JOIN product_variants v ON v.tenant_id = m.tenant_id AND v.id = m.variant_id
        GROUP BY m.tenant_id, v.product_id
    ) a ON a.tenant_id = b.tenant_id AND a.product_id = b.product_id
    WHERE a.quantity IS DISTINCT FROM b.quantity OR a.movements IS DISTINCT FROM b.movements;
    IF kardex_changed <> 0 THEN
        RAISE EXCEPTION 'stock by variant: % products whose kardex changed', kardex_changed;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE stock_levels FORCE ROW LEVEL SECURITY;
ALTER TABLE stock_movements FORCE ROW LEVEL SECURITY;
ALTER TABLE product_variants FORCE ROW LEVEL SECURITY;

-- The lot of the stock; NULL for stock without a lot. Its table and foreign
-- key arrive with the lots_expiry module.
ALTER TABLE stock_levels ADD COLUMN lot_id uuid;
ALTER TABLE stock_movements ADD COLUMN lot_id uuid;

ALTER TABLE stock_levels DROP CONSTRAINT stock_levels_pkey;
-- EnsureStockLevel's ON CONFLICT uses this key.
ALTER TABLE stock_levels ADD CONSTRAINT stock_levels_key
    UNIQUE NULLS NOT DISTINCT (tenant_id, variant_id, location_id, lot_id);

-- The adapter maps violations of these constraints (by name) to domain errors.
ALTER TABLE stock_levels ADD CONSTRAINT stock_levels_variant_fkey
    FOREIGN KEY (tenant_id, variant_id) REFERENCES product_variants (tenant_id, id);
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_variant_fkey
    FOREIGN KEY (tenant_id, variant_id) REFERENCES product_variants (tenant_id, id);

-- The index names keep the old column out of them.
ALTER INDEX stock_levels_tenant_id_location_id_idx RENAME TO stock_levels_location_idx;

-- +goose Down
-- Fails if there is stock with a lot, or if two variants of a product have
-- stock at the same location: a product cannot hold both.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM stock_levels WHERE lot_id IS NOT NULL)
       OR EXISTS (SELECT 1 FROM stock_movements WHERE lot_id IS NOT NULL) THEN
        RAISE EXCEPTION 'stock by variant down: there is stock with a lot';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER INDEX stock_levels_location_idx RENAME TO stock_levels_tenant_id_location_id_idx;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_variant_fkey;
ALTER TABLE stock_levels DROP CONSTRAINT stock_levels_variant_fkey;
ALTER TABLE stock_levels DROP CONSTRAINT stock_levels_key;
ALTER TABLE stock_movements DROP COLUMN lot_id;
ALTER TABLE stock_levels DROP COLUMN lot_id;

ALTER TABLE stock_levels NO FORCE ROW LEVEL SECURITY;
ALTER TABLE stock_movements NO FORCE ROW LEVEL SECURITY;
ALTER TABLE product_variants NO FORCE ROW LEVEL SECURITY;

-- Each variant's rows go back to its product.
UPDATE stock_levels l
SET variant_id = v.product_id
FROM product_variants v
WHERE v.tenant_id = l.tenant_id AND v.id = l.variant_id AND v.id <> v.product_id;

UPDATE stock_movements m
SET variant_id = v.product_id
FROM product_variants v
WHERE v.tenant_id = m.tenant_id AND v.id = m.variant_id AND v.id <> v.product_id;

ALTER TABLE stock_levels FORCE ROW LEVEL SECURITY;
ALTER TABLE stock_movements FORCE ROW LEVEL SECURITY;
ALTER TABLE product_variants FORCE ROW LEVEL SECURITY;

ALTER TABLE stock_levels RENAME COLUMN variant_id TO product_id;
ALTER TABLE stock_movements RENAME COLUMN variant_id TO product_id;
ALTER TABLE stock_levels ADD PRIMARY KEY (tenant_id, product_id, location_id);
ALTER TABLE stock_levels ADD CONSTRAINT stock_levels_product_fkey
    FOREIGN KEY (tenant_id, product_id) REFERENCES products (tenant_id, id);
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_product_fkey
    FOREIGN KEY (tenant_id, product_id) REFERENCES products (tenant_id, id);
