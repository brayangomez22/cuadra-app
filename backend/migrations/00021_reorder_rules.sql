-- Inventory: the minimum and maximum stock of a variant at a location, in the
-- product's base unit. When the stock (of every lot) falls to the minimum or
-- below, the variant shows up in the reorder alerts with how much to order to
-- reach the maximum. A variant without a rule raises no alerts.
--
-- A separate table and not columns of stock_levels: a variant that never had
-- a movement has no level, and with a rule and no stock it must still show up.

-- +goose Up
CREATE TABLE reorder_rules (
    tenant_id    uuid          NOT NULL REFERENCES tenants (id),
    variant_id   uuid          NOT NULL,
    location_id  uuid          NOT NULL,
    min_quantity numeric(18,4) NOT NULL,
    max_quantity numeric(18,4) NOT NULL,
    updated_at   timestamptz   NOT NULL,
    PRIMARY KEY (tenant_id, location_id, variant_id),
    CONSTRAINT reorder_rules_range CHECK (min_quantity >= 0 AND max_quantity >= min_quantity),
    -- The adapter maps a violation of these constraints (by name) to
    -- ErrVariantNotFound and ErrLocationNotFound.
    CONSTRAINT reorder_rules_variant_fkey FOREIGN KEY (tenant_id, variant_id)
        REFERENCES product_variants (tenant_id, id),
    CONSTRAINT reorder_rules_location_fkey FOREIGN KEY (tenant_id, location_id)
        REFERENCES locations (tenant_id, id)
);

ALTER TABLE reorder_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE reorder_rules FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON reorder_rules
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- +goose Down
DROP TABLE reorder_rules;
