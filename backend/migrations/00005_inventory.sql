-- Inventory: locations, stock levels, the kardex (stock movements) and the
-- tenant's stock settings, isolated per tenant with Row-Level Security.
--
-- Foreign keys include tenant_id (Postgres checks them without Row-Level
-- Security), so a level or a movement cannot point to another tenant's
-- product or location. stock_movements is append-only for the API: app_user
-- may insert and read it, never update or delete it.

-- +goose Up

-- Target of the tenant-scoped foreign keys to products below.
ALTER TABLE products ADD CONSTRAINT products_tenant_id_id_key UNIQUE (tenant_id, id);

CREATE TABLE locations (
    id         uuid        PRIMARY KEY,
    tenant_id  uuid        NOT NULL REFERENCES tenants (id),
    name       text        NOT NULL CHECK (name <> ''),
    active     boolean     NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT locations_tenant_id_id_key UNIQUE (tenant_id, id)
);

-- "Bodega" and "bodega" are the same location. The adapter maps a violation
-- of this index (by name) to ErrLocationNameTaken.
CREATE UNIQUE INDEX locations_tenant_id_name_key ON locations (tenant_id, lower(name));

ALTER TABLE locations ENABLE ROW LEVEL SECURITY;
ALTER TABLE locations FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON locations
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- The current stock of a product at a location. A row exists once the product
-- has had a movement there; no row means zero. The repository locks it
-- (SELECT ... FOR UPDATE) before every movement.
CREATE TABLE stock_levels (
    tenant_id    uuid          NOT NULL REFERENCES tenants (id),
    product_id   uuid          NOT NULL,
    location_id  uuid          NOT NULL,
    -- Negative only if the tenant allows selling without stock.
    quantity     numeric(18,4) NOT NULL,
    average_cost numeric(18,4) NOT NULL CHECK (average_cost >= 0),
    updated_at   timestamptz   NOT NULL,
    PRIMARY KEY (tenant_id, product_id, location_id),
    -- The adapter maps violations of these constraints (by name) to domain errors.
    CONSTRAINT stock_levels_product_fkey FOREIGN KEY (tenant_id, product_id) REFERENCES products (tenant_id, id),
    CONSTRAINT stock_levels_location_fkey FOREIGN KEY (tenant_id, location_id) REFERENCES locations (tenant_id, id)
);

CREATE INDEX stock_levels_tenant_id_location_id_idx ON stock_levels (tenant_id, location_id, product_id);

ALTER TABLE stock_levels ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_levels FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON stock_levels
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- The kardex. Quantity is signed (positive in, negative out); balance_after
-- and average_cost_after are the level right after the movement.
CREATE TABLE stock_movements (
    id                 uuid          PRIMARY KEY,
    tenant_id          uuid          NOT NULL REFERENCES tenants (id),
    product_id         uuid          NOT NULL,
    location_id        uuid          NOT NULL,
    type               text          NOT NULL CHECK (type IN ('purchase_in', 'sale_out', 'adjustment', 'transfer_out', 'transfer_in')),
    quantity           numeric(18,4) NOT NULL,
    unit_cost          numeric(18,4) NOT NULL CHECK (unit_cost >= 0),
    balance_after      numeric(18,4) NOT NULL,
    average_cost_after numeric(18,4) NOT NULL CHECK (average_cost_after >= 0),
    -- Required for adjustments, empty for the other types.
    reason             text          NOT NULL,
    reference_kind     text          CHECK (reference_kind IN ('purchase', 'sale', 'transfer')),
    reference_id       uuid,
    user_id            uuid          NOT NULL,
    occurred_at        timestamptz   NOT NULL,
    CONSTRAINT stock_movements_quantity_sign CHECK (CASE
        WHEN type IN ('purchase_in', 'transfer_in') THEN quantity > 0
        WHEN type IN ('sale_out', 'transfer_out') THEN quantity < 0
        ELSE quantity <> 0
    END),
    CONSTRAINT stock_movements_reason CHECK ((type = 'adjustment') = (reason <> '')),
    CONSTRAINT stock_movements_reference CHECK ((reference_kind IS NULL) = (reference_id IS NULL)),
    -- Transfers, and only transfers, carry a transfer reference.
    CONSTRAINT stock_movements_transfer_reference CHECK (
        (type IN ('transfer_out', 'transfer_in')) = coalesce(reference_kind = 'transfer', false)
    ),
    CONSTRAINT stock_movements_product_fkey FOREIGN KEY (tenant_id, product_id) REFERENCES products (tenant_id, id),
    CONSTRAINT stock_movements_location_fkey FOREIGN KEY (tenant_id, location_id) REFERENCES locations (tenant_id, id)
);

-- A product's kardex, newest first.
CREATE INDEX stock_movements_kardex_idx ON stock_movements (tenant_id, product_id, occurred_at DESC, id DESC);

ALTER TABLE stock_movements ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_movements FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON stock_movements
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- Movements are immutable: a mistake is corrected with another movement.
REVOKE UPDATE, DELETE ON stock_movements FROM app_user;

-- The tenant's stock rules. No row means the defaults (no negative stock).
CREATE TABLE inventory_settings (
    tenant_id            uuid    PRIMARY KEY REFERENCES tenants (id),
    allow_negative_stock boolean NOT NULL DEFAULT false
);

ALTER TABLE inventory_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory_settings FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON inventory_settings
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- +goose Down
DROP TABLE inventory_settings;
DROP TABLE stock_movements;
DROP TABLE stock_levels;
DROP TABLE locations;
ALTER TABLE products DROP CONSTRAINT products_tenant_id_id_key;
