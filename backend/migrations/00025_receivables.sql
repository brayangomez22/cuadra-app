-- Receivables: the credit a tenant gives its customers. A credit account
-- holds a customer's limit, term and balance; each credit sale opens a
-- receivable, due the term's days after the day of the sale; payments are
-- split among the receivables. Isolated per tenant with Row-Level Security.
--
-- The account's balance is the sum of its open receivables' balances. The
-- application keeps it so, locking the account row for every charge, payment
-- and cancellation; the CHECKs keep amounts and balances within bounds.
--
-- Sales and cash sessions accept the new payment method "credit": the part
-- of a sale left to pay later. It gives no change and is not cash.
--
-- Foreign keys include tenant_id (Postgres checks them without Row-Level
-- Security), so nothing points to another tenant's rows. User ids have no
-- foreign key, like the rest of the tables.

-- +goose Up
CREATE TABLE credit_accounts (
    tenant_id    uuid          NOT NULL REFERENCES tenants (id),
    customer_id  uuid          NOT NULL,
    credit_limit numeric(18,4) NOT NULL CHECK (credit_limit >= 0),
    term_days    integer       NOT NULL CHECK (term_days BETWEEN 0 AND 365),
    balance      numeric(18,4) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at   timestamptz   NOT NULL,
    PRIMARY KEY (tenant_id, customer_id),
    CONSTRAINT credit_accounts_customer_fkey FOREIGN KEY (tenant_id, customer_id)
        REFERENCES customers (tenant_id, id)
);

CREATE TABLE receivables (
    id            uuid          PRIMARY KEY,
    tenant_id     uuid          NOT NULL REFERENCES tenants (id),
    customer_id   uuid          NOT NULL,
    sale_id       uuid          NOT NULL,
    amount        numeric(18,4) NOT NULL CHECK (amount > 0),
    balance       numeric(18,4) NOT NULL CHECK (balance >= 0 AND balance <= amount),
    due_date      date          NOT NULL,
    status        text          NOT NULL CHECK (status IN ('open', 'paid', 'canceled')),
    authorized_by uuid,
    authorized_at timestamptz,
    created_at    timestamptz   NOT NULL,
    canceled_at  timestamptz,
    CONSTRAINT receivables_tenant_id_id_key UNIQUE (tenant_id, id),
    -- A sale opens at most one receivable.
    CONSTRAINT receivables_sale_key UNIQUE (tenant_id, sale_id),
    CONSTRAINT receivables_account_fkey FOREIGN KEY (tenant_id, customer_id)
        REFERENCES credit_accounts (tenant_id, customer_id),
    -- Deferred: a credit sale opens its receivable before the sale's row is
    -- inserted, in the same transaction.
    CONSTRAINT receivables_sale_fkey FOREIGN KEY (tenant_id, sale_id) REFERENCES sales (tenant_id, id)
        DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT receivables_authorized CHECK ((authorized_by IS NULL) = (authorized_at IS NULL)),
    CONSTRAINT receivables_open_balance CHECK ((status = 'open') = (balance > 0)),
    CONSTRAINT receivables_canceled CHECK ((status = 'canceled') = (canceled_at IS NOT NULL))
);

CREATE INDEX receivables_open_idx ON receivables (tenant_id, customer_id, due_date) WHERE status = 'open';
CREATE INDEX receivables_customer_idx ON receivables (tenant_id, customer_id, created_at);

CREATE TABLE customer_payments (
    id              uuid          PRIMARY KEY,
    tenant_id       uuid          NOT NULL REFERENCES tenants (id),
    customer_id     uuid          NOT NULL,
    method          text          NOT NULL CHECK (method IN ('cash', 'card', 'nequi', 'daviplata', 'transfer')),
    amount          numeric(18,4) NOT NULL CHECK (amount > 0),
    user_id         uuid          NOT NULL,
    cash_session_id uuid,
    created_at      timestamptz   NOT NULL,
    CONSTRAINT customer_payments_tenant_id_id_key UNIQUE (tenant_id, id),
    CONSTRAINT customer_payments_account_fkey FOREIGN KEY (tenant_id, customer_id)
        REFERENCES credit_accounts (tenant_id, customer_id),
    CONSTRAINT customer_payments_cash_session_fkey FOREIGN KEY (tenant_id, cash_session_id)
        REFERENCES cash_sessions (tenant_id, id)
);

CREATE INDEX customer_payments_customer_idx ON customer_payments (tenant_id, customer_id, created_at);

CREATE TABLE customer_payment_allocations (
    tenant_id     uuid          NOT NULL REFERENCES tenants (id),
    payment_id    uuid          NOT NULL,
    receivable_id uuid          NOT NULL,
    position      integer       NOT NULL CHECK (position >= 0),
    amount        numeric(18,4) NOT NULL CHECK (amount > 0),
    PRIMARY KEY (tenant_id, payment_id, receivable_id),
    CONSTRAINT customer_payment_allocations_payment_fkey FOREIGN KEY (tenant_id, payment_id)
        REFERENCES customer_payments (tenant_id, id),
    CONSTRAINT customer_payment_allocations_receivable_fkey FOREIGN KEY (tenant_id, receivable_id)
        REFERENCES receivables (tenant_id, id)
);

ALTER TABLE credit_accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE credit_accounts FORCE ROW LEVEL SECURITY;
ALTER TABLE receivables ENABLE ROW LEVEL SECURITY;
ALTER TABLE receivables FORCE ROW LEVEL SECURITY;
ALTER TABLE customer_payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_payments FORCE ROW LEVEL SECURITY;
ALTER TABLE customer_payment_allocations ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_payment_allocations FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON credit_accounts
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());
CREATE POLICY tenant_isolation ON receivables
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());
CREATE POLICY tenant_isolation ON customer_payments
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());
CREATE POLICY tenant_isolation ON customer_payment_allocations
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

ALTER TABLE sale_payments DROP CONSTRAINT sale_payments_method_check;
ALTER TABLE sale_payments ADD CONSTRAINT sale_payments_method_check
    CHECK (method IN ('cash', 'card', 'nequi', 'daviplata', 'transfer', 'credit'));
ALTER TABLE cash_session_payments DROP CONSTRAINT cash_session_payments_method_check;
ALTER TABLE cash_session_payments ADD CONSTRAINT cash_session_payments_method_check
    CHECK (method IN ('cash', 'card', 'nequi', 'daviplata', 'transfer', 'credit'));

-- +goose Down
-- A credit sale has no method to go back to: going down with any is refused
-- rather than rewriting the sales and the closed cash sessions. The owner
-- reads every tenant's payments with Row-Level Security not forced.
ALTER TABLE sale_payments NO FORCE ROW LEVEL SECURITY;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM sale_payments WHERE method = 'credit') THEN
        RAISE EXCEPTION 'there are credit sales: cannot go back to sales without credit';
    END IF;
END
$$;
-- +goose StatementEnd
ALTER TABLE sale_payments FORCE ROW LEVEL SECURITY;

ALTER TABLE cash_session_payments DROP CONSTRAINT cash_session_payments_method_check;
ALTER TABLE cash_session_payments ADD CONSTRAINT cash_session_payments_method_check
    CHECK (method IN ('cash', 'card', 'nequi', 'daviplata', 'transfer'));
ALTER TABLE sale_payments DROP CONSTRAINT sale_payments_method_check;
ALTER TABLE sale_payments ADD CONSTRAINT sale_payments_method_check
    CHECK (method IN ('cash', 'card', 'nequi', 'daviplata', 'transfer'));

DROP TABLE customer_payment_allocations;
DROP TABLE customer_payments;
DROP TABLE receivables;
DROP TABLE credit_accounts;
