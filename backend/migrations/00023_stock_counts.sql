-- Inventory: physical stock counts (T30) and the initial stock movement type
-- (P81).
--
-- A count is a session at one location, of every variant with stock there
-- (full) or of the variants of some categories (partial). Its lines are the
-- snapshot of the stock when it started; its entries are what each person
-- counted. The business keeps selling meanwhile: the expected quantity of a
-- line is its stock at the start plus the movements of the kardex up to its
-- last entry. Approving it records one adjustment per line with a
-- difference, referencing the count.
--
-- initial_in is the initial stock a business loads (an import): it enters
-- and sets the average cost like purchase_in, but it is not a purchase.

-- +goose Up
CREATE TABLE stock_counts (
    id            uuid        PRIMARY KEY,
    tenant_id     uuid        NOT NULL REFERENCES tenants (id),
    location_id   uuid        NOT NULL,
    -- Empty for a full count.
    category_ids  uuid[]      NOT NULL DEFAULT '{}',
    mode          text        NOT NULL CHECK (mode IN ('sum', 'review')),
    status        text        NOT NULL CHECK (status IN ('open', 'approved', 'canceled')),
    started_by    uuid        NOT NULL,
    started_at    timestamptz NOT NULL,
    closed_by     uuid,
    closed_at     timestamptz,
    -- The accuracy of an approved count, kept for the report.
    lines_total   integer,
    lines_counted integer,
    lines_exact   integer,
    net_difference_value      numeric(18,4),
    absolute_difference_value numeric(18,4),
    CONSTRAINT stock_counts_closed CHECK ((status = 'open') = (closed_at IS NULL)),
    CONSTRAINT stock_counts_closed_by CHECK ((closed_at IS NULL) = (closed_by IS NULL)),
    CONSTRAINT stock_counts_summary CHECK ((status = 'approved') = (lines_counted IS NOT NULL)),
    -- The adapter maps a violation of this constraint (by name) to
    -- ErrLocationNotFound.
    CONSTRAINT stock_counts_location_fkey FOREIGN KEY (tenant_id, location_id)
        REFERENCES locations (tenant_id, id),
    UNIQUE (tenant_id, id)
);

-- One open count per location: two would adjust the same stock twice. The
-- adapter maps a violation (by name) to ErrCountInProgress.
CREATE UNIQUE INDEX stock_counts_one_open ON stock_counts (tenant_id, location_id) WHERE status = 'open';
CREATE INDEX stock_counts_list_idx ON stock_counts (tenant_id, started_at DESC, id DESC);

-- The snapshot: one line per variant in the count. round goes up with every
-- recount; only the entries of the current round count.
CREATE TABLE stock_count_lines (
    tenant_id      uuid          NOT NULL REFERENCES tenants (id),
    count_id       uuid          NOT NULL,
    variant_id     uuid          NOT NULL,
    start_quantity numeric(18,4) NOT NULL,
    start_cost     numeric(18,4) NOT NULL CHECK (start_cost >= 0),
    round          integer       NOT NULL DEFAULT 1 CHECK (round >= 1),
    PRIMARY KEY (tenant_id, count_id, variant_id),
    CONSTRAINT stock_count_lines_count_fkey FOREIGN KEY (tenant_id, count_id)
        REFERENCES stock_counts (tenant_id, id),
    -- The adapter maps a violation of this constraint (by name) to
    -- ErrVariantNotFound.
    CONSTRAINT stock_count_lines_variant_fkey FOREIGN KEY (tenant_id, variant_id)
        REFERENCES product_variants (tenant_id, id)
);

-- What one person counted of a variant, in its base unit.
CREATE TABLE stock_count_entries (
    id         uuid          PRIMARY KEY,
    tenant_id  uuid          NOT NULL REFERENCES tenants (id),
    count_id   uuid          NOT NULL,
    variant_id uuid          NOT NULL,
    round      integer       NOT NULL,
    quantity   numeric(18,4) NOT NULL CHECK (quantity >= 0),
    counted_by uuid          NOT NULL,
    counted_at timestamptz   NOT NULL,
    CONSTRAINT stock_count_entries_line_fkey FOREIGN KEY (tenant_id, count_id, variant_id)
        REFERENCES stock_count_lines (tenant_id, count_id, variant_id)
);

CREATE INDEX stock_count_entries_line_idx ON stock_count_entries (tenant_id, count_id, variant_id, round);

ALTER TABLE stock_counts ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_counts FORCE ROW LEVEL SECURITY;
ALTER TABLE stock_count_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_count_lines FORCE ROW LEVEL SECURITY;
ALTER TABLE stock_count_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_count_entries FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON stock_counts
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());
CREATE POLICY tenant_isolation ON stock_count_lines
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());
CREATE POLICY tenant_isolation ON stock_count_entries
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- The adjustments of a count reference it.
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_reference_kind_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_reference_kind_check
    CHECK (reference_kind IN ('purchase', 'sale', 'transfer', 'stock_count'));

ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_type_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_type_check
    CHECK (type IN ('purchase_in', 'initial_in', 'sale_out', 'sale_void', 'adjustment', 'transfer_out', 'transfer_in'));

ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_quantity_sign;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_quantity_sign CHECK (CASE
    WHEN type IN ('purchase_in', 'initial_in', 'sale_void', 'transfer_in') THEN quantity > 0
    WHEN type IN ('sale_out', 'transfer_out') THEN quantity < 0
    ELSE quantity <> 0
END);

-- +goose Down
-- Fails if the kardex already has initial_in movements or adjustments of a
-- count: they are immutable.
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_quantity_sign;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_quantity_sign CHECK (CASE
    WHEN type IN ('purchase_in', 'sale_void', 'transfer_in') THEN quantity > 0
    WHEN type IN ('sale_out', 'transfer_out') THEN quantity < 0
    ELSE quantity <> 0
END);
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_type_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_type_check
    CHECK (type IN ('purchase_in', 'sale_out', 'sale_void', 'adjustment', 'transfer_out', 'transfer_in'));
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_reference_kind_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_reference_kind_check
    CHECK (reference_kind IN ('purchase', 'sale', 'transfer'));

DROP TABLE stock_count_entries;
DROP TABLE stock_count_lines;
DROP TABLE stock_counts;
