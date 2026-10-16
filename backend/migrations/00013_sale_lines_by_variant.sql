-- Sales: each line records the variant sold (variant_id), its label at the
-- time of the sale (variant_label, NULL for a variant without axes) and the
-- price rule that gave its unit price (price_rule, a stable text id such as
-- 'variant_price'; modules add rules, so it is not an enum). product_id
-- stays: the reports by product read it.
--
-- Until now sales sold the default variant of each product, so every
-- existing line takes the default variant of its product and the base rule.
-- The default variant of a product from before 00010 has the product's id;
-- one created after has its own. The owner may update sale_lines (app_user
-- may not), and the backfill only adds what each line already sold: no
-- amount changes, which the checks below verify per tenant.

-- +goose Up

ALTER TABLE sale_lines ADD COLUMN variant_id uuid;
ALTER TABLE sale_lines ADD COLUMN variant_label text CHECK (variant_label <> '');
ALTER TABLE sale_lines ADD COLUMN price_rule text CHECK (price_rule ~ '^[a-z][a-z0-9_]{0,63}$');

-- FORCE ROW LEVEL SECURITY applies to the owner too (see 00010): it is lifted
-- for the backfill and the checks, and restored in the same transaction.
ALTER TABLE sale_lines NO FORCE ROW LEVEL SECURITY;
ALTER TABLE product_variants NO FORCE ROW LEVEL SECURITY;

-- The lines and their amounts per tenant, to compare after the backfill.
CREATE TEMP TABLE sale_lines_before ON COMMIT DROP AS
SELECT tenant_id, count(*) AS lines, sum(quantity) AS quantity, sum(unit_price) AS unit_price,
       sum(gross) AS gross, sum(discount) AS discount, sum(tax) AS tax
FROM sale_lines GROUP BY tenant_id;

UPDATE sale_lines l
SET variant_id = v.id, price_rule = 'variant_price'
FROM product_variants v
WHERE v.tenant_id = l.tenant_id AND v.product_id = l.product_id AND v.is_default;

-- Checked before FORCE is back: with it, every count would read zero.
-- +goose StatementBegin
DO $$
DECLARE
    unmatched bigint;
    changed bigint;
BEGIN
    SELECT count(*) INTO unmatched FROM sale_lines WHERE variant_id IS NULL OR price_rule IS NULL;
    IF unmatched <> 0 THEN
        RAISE EXCEPTION 'sale lines by variant: % lines without a default variant', unmatched;
    END IF;

    SELECT count(*) INTO changed
    FROM sale_lines_before b
    FULL JOIN (
        SELECT tenant_id, count(*) AS lines, sum(quantity) AS quantity, sum(unit_price) AS unit_price,
               sum(gross) AS gross, sum(discount) AS discount, sum(tax) AS tax
        FROM sale_lines GROUP BY tenant_id
    ) a ON a.tenant_id = b.tenant_id
    WHERE a.lines IS DISTINCT FROM b.lines OR a.quantity IS DISTINCT FROM b.quantity
       OR a.unit_price IS DISTINCT FROM b.unit_price OR a.gross IS DISTINCT FROM b.gross
       OR a.discount IS DISTINCT FROM b.discount OR a.tax IS DISTINCT FROM b.tax;
    IF changed <> 0 THEN
        RAISE EXCEPTION 'sale lines by variant: % tenants whose sale lines changed', changed;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE sale_lines FORCE ROW LEVEL SECURITY;
ALTER TABLE product_variants FORCE ROW LEVEL SECURITY;

ALTER TABLE sale_lines ALTER COLUMN variant_id SET NOT NULL;
ALTER TABLE sale_lines ALTER COLUMN price_rule SET NOT NULL;

-- The line's variant belongs to the line's product: the reports by product
-- can trust product_id. A foreign key needs a unique key on the referenced
-- columns; id alone is already unique, so this one costs only its index.
ALTER TABLE product_variants ADD CONSTRAINT product_variants_tenant_id_id_product_id_key
    UNIQUE (tenant_id, id, product_id);
ALTER TABLE sale_lines ADD CONSTRAINT sale_lines_variant_fkey
    FOREIGN KEY (tenant_id, variant_id, product_id) REFERENCES product_variants (tenant_id, id, product_id);

-- +goose Down
-- Fails if a line holds what the lines before 00013 cannot say: a variant
-- other than its product's default, a label or a price rule other than the
-- variant's price.
ALTER TABLE sale_lines NO FORCE ROW LEVEL SECURITY;
ALTER TABLE product_variants NO FORCE ROW LEVEL SECURITY;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM sale_lines l JOIN product_variants v ON v.tenant_id = l.tenant_id AND v.id = l.variant_id
        WHERE NOT v.is_default OR l.variant_label IS NOT NULL OR l.price_rule <> 'variant_price'
    ) THEN
        RAISE EXCEPTION 'sale lines by variant down: there are lines the previous model cannot represent';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE sale_lines FORCE ROW LEVEL SECURITY;
ALTER TABLE product_variants FORCE ROW LEVEL SECURITY;

ALTER TABLE sale_lines DROP CONSTRAINT sale_lines_variant_fkey;
ALTER TABLE product_variants DROP CONSTRAINT product_variants_tenant_id_id_product_id_key;
ALTER TABLE sale_lines DROP COLUMN price_rule;
ALTER TABLE sale_lines DROP COLUMN variant_label;
ALTER TABLE sale_lines DROP COLUMN variant_id;
