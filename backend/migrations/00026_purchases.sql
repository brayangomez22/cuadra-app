-- Purchases: the tenant's suppliers and the purchases registered from their
-- invoices. Each line keeps the unit it was bought in, with its quantity and
-- cost in that unit (P59), and what it put into the stock in the base unit:
-- the purchase_in movements of the kardex reference the purchase. A purchase
-- on credit keeps the balance left to pay and its due date. Isolated per
-- tenant with Row-Level Security.
--
-- Foreign keys include tenant_id (Postgres checks them without Row-Level
-- Security), so nothing points to another tenant's rows. User ids have no
-- foreign key, like the rest of the tables.

-- +goose Up
CREATE TABLE suppliers (
    id                uuid        PRIMARY KEY,
    tenant_id         uuid        NOT NULL REFERENCES tenants (id),
    name              text        NOT NULL CHECK (name <> ''),
    -- The NIT's number and check digit, without separators: "900123456-8".
    nit               text        NOT NULL CHECK (nit ~ '^[0-9]{3,15}-[0-9]$'),
    contact_name      text,
    email             text,
    phone             text,
    payment_term_days integer     NOT NULL CHECK (payment_term_days BETWEEN 0 AND 365),
    active            boolean     NOT NULL,
    -- Lowercase and without accents, like customers.search_name.
    search_name       text        NOT NULL GENERATED ALWAYS AS (public.f_unaccent(lower(name))) STORED,
    created_at        timestamptz NOT NULL,
    updated_at        timestamptz NOT NULL,
    CONSTRAINT suppliers_tenant_id_id_key UNIQUE (tenant_id, id),
    -- The adapter maps a violation of this constraint (by name) to ErrNITTaken.
    CONSTRAINT suppliers_nit_key UNIQUE (tenant_id, nit)
);

CREATE TABLE purchases (
    id             uuid          PRIMARY KEY,
    tenant_id      uuid          NOT NULL REFERENCES tenants (id),
    supplier_id    uuid          NOT NULL,
    location_id    uuid          NOT NULL,
    invoice_number text          NOT NULL CHECK (invoice_number <> ''),
    -- The number in capitals and without spaces, dashes or dots: a supplier's
    -- invoice is registered once.
    invoice_key    text          NOT NULL CHECK (invoice_key <> ''),
    issued_on      date          NOT NULL,
    due_on         date,
    payment        text          NOT NULL CHECK (payment IN ('cash', 'credit')),
    notes          text,
    subtotal       numeric(18,4) NOT NULL CHECK (subtotal >= 0),
    tax            numeric(18,4) NOT NULL CHECK (tax >= 0),
    total          numeric(18,4) NOT NULL CHECK (total = subtotal + tax),
    balance_due    numeric(18,4) NOT NULL CHECK (balance_due >= 0 AND balance_due <= total),
    created_by     uuid          NOT NULL,
    created_at     timestamptz   NOT NULL,
    CONSTRAINT purchases_tenant_id_id_key UNIQUE (tenant_id, id),
    -- The adapter maps a violation of this constraint (by name) to ErrInvoiceTaken.
    CONSTRAINT purchases_invoice_key UNIQUE (tenant_id, supplier_id, invoice_key),
    CONSTRAINT purchases_supplier_fkey FOREIGN KEY (tenant_id, supplier_id) REFERENCES suppliers (tenant_id, id),
    CONSTRAINT purchases_location_fkey FOREIGN KEY (tenant_id, location_id) REFERENCES locations (tenant_id, id),
    -- Only a purchase on credit falls due, never before its invoice; one in
    -- cash owes nothing.
    CONSTRAINT purchases_due CHECK (
        (payment = 'credit' AND due_on IS NOT NULL AND due_on >= issued_on)
        OR (payment = 'cash' AND due_on IS NULL AND balance_due = 0))
);

CREATE INDEX purchases_issued_idx ON purchases (tenant_id, issued_on DESC, created_at DESC);
CREATE INDEX purchases_open_idx ON purchases (tenant_id, supplier_id) WHERE balance_due > 0;

CREATE TABLE purchase_lines (
    tenant_id      uuid          NOT NULL REFERENCES tenants (id),
    purchase_id    uuid          NOT NULL,
    position       integer       NOT NULL CHECK (position >= 0),
    variant_id     uuid          NOT NULL,
    product_name   text          NOT NULL CHECK (product_name <> ''),
    variant_label  text,
    -- The alternate unit it was bought in, or NULL for the base unit (factor 1).
    unit_id        uuid,
    unit_code      text,
    unit_factor    numeric(18,4) NOT NULL CHECK (unit_factor > 0),
    quantity       numeric(18,4) NOT NULL CHECK (quantity > 0),
    unit_cost      numeric(18,4) NOT NULL CHECK (unit_cost >= 0),
    tax_rate       numeric(18,4) NOT NULL CHECK (tax_rate >= 0 AND tax_rate < 1),
    subtotal       numeric(18,4) NOT NULL CHECK (subtotal >= 0),
    tax            numeric(18,4) NOT NULL CHECK (tax >= 0),
    base_quantity  numeric(18,4) NOT NULL CHECK (base_quantity > 0),
    base_unit_cost numeric(18,4) NOT NULL CHECK (base_unit_cost >= 0),
    -- Optional, like the movements': lots arrive with the lots_expiry module.
    lot_id         uuid,
    PRIMARY KEY (tenant_id, purchase_id, position),
    CONSTRAINT purchase_lines_purchase_fkey FOREIGN KEY (tenant_id, purchase_id) REFERENCES purchases (tenant_id, id),
    CONSTRAINT purchase_lines_variant_fkey FOREIGN KEY (tenant_id, variant_id)
        REFERENCES product_variants (tenant_id, id),
    CONSTRAINT purchase_lines_unit_fkey FOREIGN KEY (tenant_id, unit_id) REFERENCES units_of_measure (tenant_id, id),
    CONSTRAINT purchase_lines_base_unit CHECK (unit_id IS NOT NULL OR unit_factor = 1)
);

CREATE INDEX purchase_lines_variant_idx ON purchase_lines (tenant_id, variant_id);

ALTER TABLE suppliers ENABLE ROW LEVEL SECURITY;
ALTER TABLE suppliers FORCE ROW LEVEL SECURITY;
ALTER TABLE purchases ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchases FORCE ROW LEVEL SECURITY;
ALTER TABLE purchase_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_lines FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON suppliers
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());
CREATE POLICY tenant_isolation ON purchases
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());
CREATE POLICY tenant_isolation ON purchase_lines
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- +goose Down
DROP TABLE purchase_lines;
DROP TABLE purchases;
DROP TABLE suppliers;
