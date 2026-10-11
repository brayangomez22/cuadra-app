-- Sales: confirmed sales with their lines and payments, the consecutive
-- number of each location and the tenant's sales settings, isolated per
-- tenant with Row-Level Security. Also lets the kardex record the stock a
-- voided sale gives back (sale_void).
--
-- Foreign keys include tenant_id (Postgres checks them without Row-Level
-- Security), so a sale cannot point to another tenant's location or product.
-- Lines and payments are immutable for the API, and sales are never deleted:
-- voiding a sale only updates its status.

-- +goose Up

CREATE TABLE sales (
    id           uuid          PRIMARY KEY,
    tenant_id    uuid          NOT NULL REFERENCES tenants (id),
    location_id  uuid          NOT NULL,
    -- Consecutive per location, without gaps (see sale_sequences).
    number       bigint        NOT NULL CHECK (number > 0),
    user_id      uuid          NOT NULL,
    -- Optional; the foreign key to customers comes with them (T31).
    customer_id  uuid,
    status       text          NOT NULL CHECK (status IN ('confirmed', 'voided')),
    -- Totals of the lines, stored so the day's list does not read them; the
    -- repository checks they add up when it reads a sale.
    subtotal     numeric(18,4) NOT NULL CHECK (subtotal >= 0),
    discounts    numeric(18,4) NOT NULL CHECK (discounts >= 0),
    tax          numeric(18,4) NOT NULL CHECK (tax >= 0),
    total        numeric(18,4) NOT NULL CHECK (total >= 0),
    paid         numeric(18,4) NOT NULL,
    change       numeric(18,4) NOT NULL CHECK (change >= 0),
    created_at   timestamptz   NOT NULL,
    confirmed_at timestamptz   NOT NULL,
    voided_at    timestamptz,
    voided_by    uuid,
    void_reason  text,
    CONSTRAINT sales_paid CHECK (paid >= total),
    CONSTRAINT sales_tenant_id_id_key UNIQUE (tenant_id, id),
    CONSTRAINT sales_number_key UNIQUE (tenant_id, location_id, number),
    CONSTRAINT sales_location_fkey FOREIGN KEY (tenant_id, location_id) REFERENCES locations (tenant_id, id),
    -- A voided sale has all the void fields; any other sale has none.
    CONSTRAINT sales_void CHECK (
        CASE WHEN status = 'voided'
            THEN voided_at IS NOT NULL AND voided_by IS NOT NULL AND void_reason IS NOT NULL AND void_reason <> ''
            ELSE voided_at IS NULL AND voided_by IS NULL AND void_reason IS NULL
        END
    )
);

-- The day's sales, newest first.
CREATE INDEX sales_confirmed_at_idx ON sales (tenant_id, confirmed_at DESC, id DESC);

ALTER TABLE sales ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sales
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

REVOKE DELETE ON sales FROM app_user;

-- The lines of a sale, in the order they were added. Name, price and rate are
-- the catalog's at the time of the sale.
CREATE TABLE sale_lines (
    tenant_id    uuid          NOT NULL REFERENCES tenants (id),
    sale_id      uuid          NOT NULL,
    position     integer       NOT NULL CHECK (position >= 0),
    product_id   uuid          NOT NULL,
    product_name text          NOT NULL CHECK (product_name <> ''),
    quantity     numeric(18,4) NOT NULL CHECK (quantity > 0),
    unit_price   numeric(18,4) NOT NULL CHECK (unit_price > 0),
    tax_rate     numeric(18,4) NOT NULL CHECK (tax_rate >= 0 AND tax_rate < 1),
    discount     numeric(18,4) NOT NULL CHECK (discount >= 0),
    gross        numeric(18,4) NOT NULL,
    tax          numeric(18,4) NOT NULL CHECK (tax >= 0),
    CONSTRAINT sale_lines_discount CHECK (discount <= gross),
    PRIMARY KEY (tenant_id, sale_id, position),
    CONSTRAINT sale_lines_sale_fkey FOREIGN KEY (tenant_id, sale_id) REFERENCES sales (tenant_id, id),
    CONSTRAINT sale_lines_product_fkey FOREIGN KEY (tenant_id, product_id) REFERENCES products (tenant_id, id)
);

ALTER TABLE sale_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE sale_lines FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sale_lines
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

REVOKE UPDATE, DELETE ON sale_lines FROM app_user;

CREATE TABLE sale_payments (
    tenant_id uuid          NOT NULL REFERENCES tenants (id),
    sale_id   uuid          NOT NULL,
    position  integer       NOT NULL CHECK (position >= 0),
    method    text          NOT NULL CHECK (method IN ('cash', 'card', 'nequi', 'daviplata', 'transfer')),
    amount    numeric(18,4) NOT NULL CHECK (amount > 0),
    PRIMARY KEY (tenant_id, sale_id, position),
    CONSTRAINT sale_payments_sale_fkey FOREIGN KEY (tenant_id, sale_id) REFERENCES sales (tenant_id, id)
);

ALTER TABLE sale_payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE sale_payments FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sale_payments
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

REVOKE UPDATE, DELETE ON sale_payments FROM app_user;

-- The last number given at each location. The repository increments it in
-- the transaction that stores the sale: the row lock serializes the sales of
-- a location and a rollback undoes the increment, so there are no gaps.
CREATE TABLE sale_sequences (
    tenant_id   uuid   NOT NULL REFERENCES tenants (id),
    location_id uuid   NOT NULL,
    last_number bigint NOT NULL CHECK (last_number > 0),
    PRIMARY KEY (tenant_id, location_id),
    CONSTRAINT sale_sequences_location_fkey FOREIGN KEY (tenant_id, location_id) REFERENCES locations (tenant_id, id)
);

ALTER TABLE sale_sequences ENABLE ROW LEVEL SECURITY;
ALTER TABLE sale_sequences FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sale_sequences
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

REVOKE DELETE ON sale_sequences FROM app_user;

-- The tenant's sales rules. No row means the defaults (no discount without
-- the sales:discount permission).
CREATE TABLE sales_settings (
    tenant_id    uuid         PRIMARY KEY REFERENCES tenants (id),
    max_discount numeric(5,4) NOT NULL DEFAULT 0 CHECK (max_discount >= 0 AND max_discount <= 1)
);

ALTER TABLE sales_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_settings FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sales_settings
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- sale_void: the stock a voided sale gives back. It enters (positive) and
-- always references the sale.
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_type_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_type_check
    CHECK (type IN ('purchase_in', 'sale_out', 'sale_void', 'adjustment', 'transfer_out', 'transfer_in'));

ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_quantity_sign;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_quantity_sign CHECK (CASE
    WHEN type IN ('purchase_in', 'sale_void', 'transfer_in') THEN quantity > 0
    WHEN type IN ('sale_out', 'transfer_out') THEN quantity < 0
    ELSE quantity <> 0
END);

ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_sale_void_reference
    CHECK (type <> 'sale_void' OR coalesce(reference_kind = 'sale', false));

-- A sale's movements, to give back its stock when it is voided.
CREATE INDEX stock_movements_reference_idx ON stock_movements (tenant_id, reference_kind, reference_id)
    WHERE reference_id IS NOT NULL;

-- +goose Down
-- Fails if the kardex already has sale_void movements: they are immutable.
DROP INDEX stock_movements_reference_idx;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_sale_void_reference;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_quantity_sign;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_quantity_sign CHECK (CASE
    WHEN type IN ('purchase_in', 'transfer_in') THEN quantity > 0
    WHEN type IN ('sale_out', 'transfer_out') THEN quantity < 0
    ELSE quantity <> 0
END);
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_type_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_type_check
    CHECK (type IN ('purchase_in', 'sale_out', 'adjustment', 'transfer_out', 'transfer_in'));

DROP TABLE sales_settings;
DROP TABLE sale_sequences;
DROP TABLE sale_payments;
DROP TABLE sale_lines;
DROP TABLE sales;
