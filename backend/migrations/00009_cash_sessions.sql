-- Cash sessions (caja): a cashier's drawer at a location, from its opening
-- float to its closing with the count by denomination (arqueo). Sales are
-- charged in the cashier's open session; cash ins and outs are recorded with
-- a reason. Isolated per tenant with Row-Level Security.
--
-- A closed session is immutable: a trigger rejects any update of it, and its
-- movements, counts and payment totals can only be inserted while it is
-- open. Nothing here is ever deleted.

-- +goose Up

CREATE TABLE cash_sessions (
    id            uuid          PRIMARY KEY,
    tenant_id     uuid          NOT NULL REFERENCES tenants (id),
    location_id   uuid          NOT NULL,
    -- The cashier: only they sell in it, record its movements and close it.
    user_id       uuid          NOT NULL,
    status        text          NOT NULL CHECK (status IN ('open', 'closed')),
    opening_float numeric(18,4) NOT NULL CHECK (opening_float >= 0),
    opened_at     timestamptz   NOT NULL,
    -- The closing: set together when the session closes.
    closed_at     timestamptz,
    closed_by     uuid,
    notes         text,
    cash_received numeric(18,4) CHECK (cash_received >= 0),
    change_given  numeric(18,4) CHECK (change_given >= 0),
    cash_in       numeric(18,4) CHECK (cash_in >= 0),
    cash_out      numeric(18,4) CHECK (cash_out >= 0),
    sales_count   integer       CHECK (sales_count >= 0),
    expected      numeric(18,4),
    counted       numeric(18,4) CHECK (counted >= 0),
    difference    numeric(18,4),
    CONSTRAINT cash_sessions_tenant_id_id_key UNIQUE (tenant_id, id),
    CONSTRAINT cash_sessions_location_fkey FOREIGN KEY (tenant_id, location_id) REFERENCES locations (tenant_id, id),
    -- A closed session has its whole closing; an open one has none.
    CONSTRAINT cash_sessions_closing CHECK (
        CASE WHEN status = 'closed'
            THEN closed_at IS NOT NULL AND closed_by IS NOT NULL AND cash_received IS NOT NULL
                AND change_given IS NOT NULL AND cash_in IS NOT NULL AND cash_out IS NOT NULL
                AND sales_count IS NOT NULL AND expected IS NOT NULL AND counted IS NOT NULL
                AND difference IS NOT NULL
            ELSE closed_at IS NULL AND closed_by IS NULL AND notes IS NULL AND cash_received IS NULL
                AND change_given IS NULL AND cash_in IS NULL AND cash_out IS NULL AND sales_count IS NULL
                AND expected IS NULL AND counted IS NULL AND difference IS NULL
        END
    ),
    -- The closing adds up: the domain computes it, the database checks it.
    CONSTRAINT cash_sessions_expected CHECK (expected = opening_float + cash_received + cash_in - cash_out - change_given),
    CONSTRAINT cash_sessions_difference CHECK (difference = counted - expected)
);

-- A user has at most one open session.
CREATE UNIQUE INDEX cash_sessions_one_open_key ON cash_sessions (tenant_id, user_id) WHERE status = 'open';

ALTER TABLE cash_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE cash_sessions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cash_sessions
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

REVOKE DELETE ON cash_sessions FROM app_user;

-- An open session only changes to closed, keeping what it was opened with;
-- a closed session never changes.
-- +goose StatementBegin
CREATE FUNCTION cash_sessions_guard_update() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.status = 'closed' THEN
        RAISE EXCEPTION 'cash session % is closed and cannot change', OLD.id
            USING ERRCODE = 'object_not_in_prerequisite_state';
    END IF;
    IF NEW.id <> OLD.id OR NEW.tenant_id <> OLD.tenant_id OR NEW.location_id <> OLD.location_id
        OR NEW.user_id <> OLD.user_id OR NEW.opening_float <> OLD.opening_float OR NEW.opened_at <> OLD.opened_at THEN
        RAISE EXCEPTION 'cash session % can only be closed', OLD.id
            USING ERRCODE = 'object_not_in_prerequisite_state';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER cash_sessions_guard_update
    BEFORE UPDATE ON cash_sessions
    FOR EACH ROW EXECUTE FUNCTION cash_sessions_guard_update();

-- The rows of a session (movements, counts, payment totals) are inserted only
-- while it is open; the closing inserts its counts before closing it.
-- +goose StatementBegin
CREATE FUNCTION cash_sessions_require_open() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM cash_sessions
        WHERE tenant_id = NEW.tenant_id AND id = NEW.session_id AND status = 'open'
    ) THEN
        RAISE EXCEPTION 'cash session % is not open', NEW.session_id
            USING ERRCODE = 'object_not_in_prerequisite_state';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- Cash ins and outs of a session, with their reason.
CREATE TABLE cash_movements (
    id         uuid          PRIMARY KEY,
    tenant_id  uuid          NOT NULL REFERENCES tenants (id),
    session_id uuid          NOT NULL,
    kind       text          NOT NULL CHECK (kind IN ('in', 'out')),
    amount     numeric(18,4) NOT NULL CHECK (amount > 0),
    reason     text          NOT NULL CHECK (reason <> ''),
    user_id    uuid          NOT NULL,
    created_at timestamptz   NOT NULL,
    CONSTRAINT cash_movements_session_fkey FOREIGN KEY (tenant_id, session_id) REFERENCES cash_sessions (tenant_id, id)
);

CREATE INDEX cash_movements_session_idx ON cash_movements (tenant_id, session_id, created_at, id);

ALTER TABLE cash_movements ENABLE ROW LEVEL SECURITY;
ALTER TABLE cash_movements FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cash_movements
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

REVOKE UPDATE, DELETE ON cash_movements FROM app_user;

CREATE TRIGGER cash_movements_require_open
    BEFORE INSERT ON cash_movements
    FOR EACH ROW EXECUTE FUNCTION cash_sessions_require_open();

-- The count of the drawer at closing, by denomination.
CREATE TABLE cash_session_counts (
    tenant_id    uuid          NOT NULL REFERENCES tenants (id),
    session_id   uuid          NOT NULL,
    denomination numeric(18,4) NOT NULL CHECK (denomination > 0),
    quantity     bigint        NOT NULL CHECK (quantity >= 0),
    PRIMARY KEY (tenant_id, session_id, denomination),
    CONSTRAINT cash_session_counts_session_fkey FOREIGN KEY (tenant_id, session_id) REFERENCES cash_sessions (tenant_id, id)
);

ALTER TABLE cash_session_counts ENABLE ROW LEVEL SECURITY;
ALTER TABLE cash_session_counts FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cash_session_counts
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

REVOKE UPDATE, DELETE ON cash_session_counts FROM app_user;

CREATE TRIGGER cash_session_counts_require_open
    BEFORE INSERT ON cash_session_counts
    FOR EACH ROW EXECUTE FUNCTION cash_sessions_require_open();

-- What the session's sales collected by payment method at closing (cash net
-- of change), for the closing report: it does not change if a sale is voided
-- afterwards.
CREATE TABLE cash_session_payments (
    tenant_id  uuid          NOT NULL REFERENCES tenants (id),
    session_id uuid          NOT NULL,
    method     text          NOT NULL CHECK (method IN ('cash', 'card', 'nequi', 'daviplata', 'transfer')),
    amount     numeric(18,4) NOT NULL CHECK (amount >= 0),
    PRIMARY KEY (tenant_id, session_id, method),
    CONSTRAINT cash_session_payments_session_fkey FOREIGN KEY (tenant_id, session_id) REFERENCES cash_sessions (tenant_id, id)
);

ALTER TABLE cash_session_payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE cash_session_payments FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON cash_session_payments
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

REVOKE UPDATE, DELETE ON cash_session_payments FROM app_user;

CREATE TRIGGER cash_session_payments_require_open
    BEFORE INSERT ON cash_session_payments
    FOR EACH ROW EXECUTE FUNCTION cash_sessions_require_open();

-- The session a sale was charged in; null for the sales before T23 and for
-- tenants that sell without one.
ALTER TABLE sales ADD COLUMN cash_session_id uuid;
ALTER TABLE sales ADD CONSTRAINT sales_cash_session_fkey
    FOREIGN KEY (tenant_id, cash_session_id) REFERENCES cash_sessions (tenant_id, id);
CREATE INDEX sales_cash_session_idx ON sales (tenant_id, cash_session_id) WHERE cash_session_id IS NOT NULL;

-- Whether users need an open cash session to sell. No settings row means yes.
ALTER TABLE sales_settings ADD COLUMN require_cash_session boolean NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE sales_settings DROP COLUMN require_cash_session;
DROP INDEX sales_cash_session_idx;
ALTER TABLE sales DROP CONSTRAINT sales_cash_session_fkey;
ALTER TABLE sales DROP COLUMN cash_session_id;
DROP TABLE cash_session_payments;
DROP TABLE cash_session_counts;
DROP TABLE cash_movements;
DROP FUNCTION cash_sessions_require_open();
DROP TABLE cash_sessions;
DROP FUNCTION cash_sessions_guard_update();
