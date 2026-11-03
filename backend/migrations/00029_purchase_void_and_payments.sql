-- Voiding a purchase registered by mistake (P100) and paying suppliers
-- (P99). A voided purchase keeps who voided it, when and why, and owes
-- nothing; its entries are taken out of the stock with purchase_void
-- movements that reference it. A payment to a supplier lowers the balance of
-- one purchase on credit; it does not touch the cash drawer. Isolated per
-- tenant with Row-Level Security. No data moves: every existing purchase
-- stays not voided and without payments.

-- +goose Up
ALTER TABLE purchases
    ADD COLUMN voided_at   timestamptz,
    ADD COLUMN voided_by   uuid,
    ADD COLUMN void_reason text;
ALTER TABLE purchases ADD CONSTRAINT purchases_void CHECK (
    (voided_at IS NULL AND voided_by IS NULL AND void_reason IS NULL)
    OR (voided_at IS NOT NULL AND voided_by IS NOT NULL AND void_reason <> '' AND balance_due = 0));

CREATE TABLE supplier_payments (
    id            uuid          PRIMARY KEY,
    tenant_id     uuid          NOT NULL REFERENCES tenants (id),
    purchase_id   uuid          NOT NULL,
    supplier_id   uuid          NOT NULL,
    method        text          NOT NULL CHECK (method IN ('cash', 'card', 'nequi', 'daviplata', 'transfer')),
    amount        numeric(18,4) NOT NULL CHECK (amount > 0),
    -- What was left to pay of the purchase right after it.
    balance_after numeric(18,4) NOT NULL CHECK (balance_after >= 0),
    paid_on       date          NOT NULL,
    created_by    uuid          NOT NULL,
    created_at    timestamptz   NOT NULL,
    CONSTRAINT supplier_payments_purchase_fkey FOREIGN KEY (tenant_id, purchase_id) REFERENCES purchases (tenant_id, id),
    CONSTRAINT supplier_payments_supplier_fkey FOREIGN KEY (tenant_id, supplier_id) REFERENCES suppliers (tenant_id, id)
);

CREATE INDEX supplier_payments_purchase_idx ON supplier_payments (tenant_id, purchase_id, created_at);

ALTER TABLE supplier_payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplier_payments FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON supplier_payments
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- purchase_void takes out what a voided purchase put in, and always
-- references it.
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_type_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_type_check
    CHECK (type IN ('purchase_in', 'initial_in', 'sale_out', 'sale_void', 'purchase_void', 'adjustment',
                    'transfer_out', 'transfer_in'));

ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_quantity_sign;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_quantity_sign CHECK (CASE
    WHEN type IN ('purchase_in', 'initial_in', 'sale_void', 'transfer_in') THEN quantity > 0
    WHEN type IN ('sale_out', 'purchase_void', 'transfer_out') THEN quantity < 0
    ELSE quantity <> 0
END);

ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_purchase_void_reference
    CHECK (type <> 'purchase_void' OR coalesce(reference_kind = 'purchase', false));

-- +goose Down
-- Fails if the kardex already has purchase_void movements: they are
-- immutable.
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_purchase_void_reference;
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_quantity_sign;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_quantity_sign CHECK (CASE
    WHEN type IN ('purchase_in', 'initial_in', 'sale_void', 'transfer_in') THEN quantity > 0
    WHEN type IN ('sale_out', 'transfer_out') THEN quantity < 0
    ELSE quantity <> 0
END);
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_type_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_type_check
    CHECK (type IN ('purchase_in', 'initial_in', 'sale_out', 'sale_void', 'adjustment', 'transfer_out', 'transfer_in'));

DROP TABLE supplier_payments;

ALTER TABLE purchases DROP CONSTRAINT purchases_void;
ALTER TABLE purchases
    DROP COLUMN void_reason,
    DROP COLUMN voided_by,
    DROP COLUMN voided_at;
