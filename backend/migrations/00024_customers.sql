-- Customers: who a tenant sells to, with what the electronic invoice needs of
-- the buyer (person type, document and tax responsibilities), and the
-- segments the tenant groups them in. Isolated per tenant with Row-Level
-- Security. Also gives sales the foreign key to customers that 00008 left
-- for this migration.
--
-- Foreign keys include tenant_id (Postgres checks them without Row-Level
-- Security), so a customer cannot point to another tenant's segment, nor a
-- sale to another tenant's customer.
--
-- Every existing tenant gets the customer segments of the ferreteria
-- template, the one 00018 gave them. A new tenant gets its template's at
-- sign-up.

-- +goose Up
CREATE TABLE customer_segments (
    id         uuid        PRIMARY KEY,
    tenant_id  uuid        NOT NULL REFERENCES tenants (id),
    name       text        NOT NULL CHECK (name <> ''),
    created_at timestamptz NOT NULL,
    -- Target of the tenant-scoped foreign key of customers.
    CONSTRAINT customer_segments_tenant_id_id_key UNIQUE (tenant_id, id)
);

-- The adapter maps a violation of this index (by name) to ErrSegmentNameTaken.
CREATE UNIQUE INDEX customer_segments_tenant_id_name_key ON customer_segments (tenant_id, lower(name));

ALTER TABLE customer_segments ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_segments FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON customer_segments
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE TABLE customers (
    id                   uuid        PRIMARY KEY,
    tenant_id            uuid        NOT NULL REFERENCES tenants (id),
    person_type          text        NOT NULL CHECK (person_type IN ('natural', 'legal')),
    document_type        text        NOT NULL CHECK (document_type IN ('cc', 'nit', 'ce', 'passport')),
    -- In the domain's canonical form: "900123456-8", "1020304050".
    document_number      text        NOT NULL CHECK (document_number <> ''),
    name                 text        NOT NULL CHECK (name <> ''),
    -- NULL when unknown, like the contact data.
    trade_name           text        CHECK (trade_name <> ''),
    email                text        CHECK (email <> ''),
    phone                text        CHECK (phone <> ''),
    address              text        CHECK (address <> ''),
    city                 text        CHECK (city <> ''),
    segment_id           uuid,
    -- DIAN codes in a fixed order; R-99-PN when none applies.
    tax_responsibilities text[]      NOT NULL CHECK (cardinality(tax_responsibilities) > 0),
    active               boolean     NOT NULL,
    -- Lowercase and without accents, so "gomez" finds "Gómez". No trigram
    -- index, as in products: under Row-Level Security a search scans the
    -- tenant's rows.
    search_name          text        NOT NULL
        GENERATED ALWAYS AS (public.f_unaccent(lower(name || ' ' || coalesce(trade_name, '')))) STORED,
    created_at           timestamptz NOT NULL,
    updated_at           timestamptz NOT NULL,
    CONSTRAINT customers_tenant_id_id_key UNIQUE (tenant_id, id),
    -- The adapter maps violations of these constraints (by name) to domain errors.
    CONSTRAINT customers_document_key UNIQUE (tenant_id, document_type, document_number),
    CONSTRAINT customers_segment_fkey FOREIGN KEY (tenant_id, segment_id) REFERENCES customer_segments (tenant_id, id),
    -- A company is identified by NIT (the domain's rule, kept by the table).
    CONSTRAINT customers_legal_person_nit CHECK (person_type = 'natural' OR document_type = 'nit')
);

CREATE INDEX customers_tenant_id_segment_id_idx ON customers (tenant_id, segment_id);

ALTER TABLE customers ENABLE ROW LEVEL SECURITY;
ALTER TABLE customers FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON customers
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- The API never accepted a customer_id before this migration, so every sale
-- has none and the constraint validates.
ALTER TABLE sales ADD CONSTRAINT sales_customer_fkey
    FOREIGN KEY (tenant_id, customer_id) REFERENCES customers (tenant_id, id);

-- Backfill. FORCE ROW LEVEL SECURITY applies to the owner too, and the owner
-- running migrations on Azure is not a superuser: with no tenant set, the
-- copy would read zero tenants and succeed. FORCE is lifted for the copy and
-- the count check, and restored in the same transaction.
ALTER TABLE tenants NO FORCE ROW LEVEL SECURITY;
ALTER TABLE customer_segments NO FORCE ROW LEVEL SECURITY;

-- The id is a UUID v7 built as in 00014 (Postgres 16 has no uuidv7()). The
-- names are those of backend/templates/business/ferreteria.yaml.
INSERT INTO customer_segments (id, tenant_id, name, created_at)
SELECT encode(
           set_bit(set_bit(
               overlay(uuid_send(gen_random_uuid())
                       PLACING substring(int8send((extract(epoch FROM clock_timestamp()) * 1000)::bigint) FROM 3)
                       FROM 1 FOR 6),
               52, 1), 53, 1),
           'hex')::uuid,
       t.id, segment.name, now()
FROM tenants t
CROSS JOIN unnest(ARRAY['Público', 'Maestro de obra', 'Contratista', 'Empresa']) AS segment (name)
WHERE t.business_template = 'ferreteria';

-- Checked before FORCE is back: with it, every count would read zero.
-- +goose StatementBegin
DO $$
DECLARE
    tenant_count   bigint;
    segment_count  bigint;
BEGIN
    SELECT count(*) INTO tenant_count FROM tenants WHERE business_template = 'ferreteria';
    SELECT count(*) INTO segment_count FROM customer_segments;
    IF segment_count <> tenant_count * 4 THEN
        RAISE EXCEPTION 'customer_segments backfill: % ferreteria tenants but % segments (expected 4 each)',
            tenant_count, segment_count;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
ALTER TABLE customer_segments FORCE ROW LEVEL SECURITY;

-- +goose Down
ALTER TABLE sales DROP CONSTRAINT sales_customer_fkey;
DROP TABLE customers;
DROP TABLE customer_segments;
