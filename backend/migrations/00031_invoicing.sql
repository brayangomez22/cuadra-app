-- Electronic documents of the DIAN (T36): the tenant's emission settings,
-- the numbering resolutions the DIAN authorized and the document each
-- confirmed sale gets. Isolated per tenant with Row-Level Security.
--
-- A resolution numbers documents of one kind with a prefix, from range_from
-- to range_to, between two days. next_number is the consecutive the next
-- document takes: the sale's transaction locks the row, takes it and adds
-- one, so numbers have no gaps and a rollback gives the number back. A
-- document's kind, prefix and number never change, and are unique: a retry
-- sends the same number and the provider is asked for it before sending
-- again, so a document is never issued twice.
--
-- A periodic job looks up the documents the DIAN has not answered across
-- tenants with tenants_with_sent_documents(), which runs as
-- invoicing_lookup and returns ids only. No data moves: every tenant starts
-- without emission.

-- +goose Up
CREATE TABLE invoicing_settings (
    tenant_id    uuid        PRIMARY KEY REFERENCES tenants (id),
    enabled      boolean     NOT NULL,
    default_kind text        NOT NULL CHECK (default_kind IN ('pos', 'invoice')),
    updated_at   timestamptz NOT NULL
);

CREATE TABLE numbering_resolutions (
    id            uuid        PRIMARY KEY,
    tenant_id     uuid        NOT NULL REFERENCES tenants (id),
    kind          text        NOT NULL CHECK (kind IN ('pos', 'invoice')),
    prefix        text        NOT NULL CHECK (prefix ~ '^[A-Z0-9]{0,4}$'),
    number        text        NOT NULL CHECK (number ~ '^[0-9]{1,30}$'),
    technical_key text        NOT NULL DEFAULT '' CHECK (char_length(technical_key) <= 100),
    range_from    bigint      NOT NULL CHECK (range_from >= 1),
    range_to      bigint      NOT NULL,
    valid_from    date        NOT NULL,
    valid_to      date        NOT NULL,
    next_number   bigint      NOT NULL,
    created_at    timestamptz NOT NULL,
    CONSTRAINT numbering_resolutions_tenant_id_id_key UNIQUE (tenant_id, id),
    CONSTRAINT numbering_resolutions_range CHECK (range_to >= range_from),
    CONSTRAINT numbering_resolutions_validity CHECK (valid_to >= valid_from),
    -- next_number passes range_to by one once the range is used up.
    CONSTRAINT numbering_resolutions_next CHECK (next_number BETWEEN range_from AND range_to + 1)
);

CREATE INDEX numbering_resolutions_kind_idx ON numbering_resolutions (tenant_id, kind);

CREATE TABLE electronic_documents (
    id            uuid        PRIMARY KEY,
    tenant_id     uuid        NOT NULL REFERENCES tenants (id),
    sale_id       uuid        NOT NULL,
    created_by    uuid        NOT NULL,
    kind          text        NOT NULL CHECK (kind IN ('pos', 'invoice')),
    resolution_id uuid        NOT NULL,
    prefix        text        NOT NULL,
    number        bigint      NOT NULL CHECK (number >= 1),
    issued_at     timestamptz NOT NULL,
    status        text        NOT NULL CHECK (status IN ('pending', 'sent', 'accepted', 'rejected')),
    attempts      integer     NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    provider_id   text        NOT NULL DEFAULT '' CHECK (char_length(provider_id) <= 200),
    code          text        NOT NULL DEFAULT '' CHECK (char_length(code) <= 200),
    qr            text        NOT NULL DEFAULT '' CHECK (char_length(qr) <= 2000),
    reasons       text[]      NOT NULL DEFAULT '{}',
    last_error    text        NOT NULL DEFAULT '' CHECK (char_length(last_error) <= 2000),
    accepted_at   timestamptz,
    updated_at    timestamptz NOT NULL,
    CONSTRAINT electronic_documents_tenant_id_id_key UNIQUE (tenant_id, id),
    -- A sale has at most one document (its credit notes are T37's).
    CONSTRAINT electronic_documents_sale_key UNIQUE (tenant_id, sale_id),
    -- The provider finds a document by its kind, prefix and number.
    CONSTRAINT electronic_documents_number_key UNIQUE (tenant_id, kind, prefix, number),
    -- Deferred: the document is stored before the sale's row is inserted,
    -- in the same transaction.
    CONSTRAINT electronic_documents_sale_fkey FOREIGN KEY (tenant_id, sale_id) REFERENCES sales (tenant_id, id)
        DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT electronic_documents_resolution_fkey FOREIGN KEY (tenant_id, resolution_id)
        REFERENCES numbering_resolutions (tenant_id, id),
    CONSTRAINT electronic_documents_accepted CHECK ((status = 'accepted') = (accepted_at IS NOT NULL AND code <> '')),
    CONSTRAINT electronic_documents_rejected CHECK ((status = 'rejected') = (cardinality(reasons) > 0))
);

CREATE INDEX electronic_documents_list_idx ON electronic_documents (tenant_id, issued_at DESC, id DESC);
CREATE INDEX electronic_documents_status_idx ON electronic_documents (tenant_id, status, issued_at DESC);

ALTER TABLE invoicing_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoicing_settings FORCE ROW LEVEL SECURITY;
ALTER TABLE numbering_resolutions ENABLE ROW LEVEL SECURITY;
ALTER TABLE numbering_resolutions FORCE ROW LEVEL SECURITY;
ALTER TABLE electronic_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE electronic_documents FORCE ROW LEVEL SECURITY;

-- The law keeps the electronic documents: the application never deletes one
-- (closing an account, T49, decides what may go).
REVOKE DELETE ON electronic_documents FROM app_user;

CREATE POLICY tenant_isolation ON invoicing_settings
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());
CREATE POLICY tenant_isolation ON numbering_resolutions
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());
CREATE POLICY tenant_isolation ON electronic_documents
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'invoicing_lookup') THEN
        CREATE ROLE invoicing_lookup NOLOGIN;
    END IF;
    -- The migration role must be a member to make invoicing_lookup the
    -- function's owner (not implied by CREATEROLE since PostgreSQL 16).
    EXECUTE format('GRANT invoicing_lookup TO %I', current_user);
END
$$;
-- +goose StatementEnd

-- Only a superuser may set SUPERUSER, BYPASSRLS and REPLICATION, even to turn
-- them off. The migration role on Azure is not one: there they are verified.
-- +goose StatementBegin
DO $$
BEGIN
    ALTER ROLE invoicing_lookup NOLOGIN NOCREATEDB NOCREATEROLE NOINHERIT;
    IF (SELECT rolsuper FROM pg_roles WHERE rolname = current_user) THEN
        ALTER ROLE invoicing_lookup NOSUPERUSER NOBYPASSRLS NOREPLICATION;
    ELSIF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'invoicing_lookup' AND (rolsuper OR rolbypassrls OR rolreplication)) THEN
        RAISE EXCEPTION 'invoicing_lookup is a superuser or has BYPASSRLS or REPLICATION: remove them as a superuser';
    END IF;
END
$$;
-- +goose StatementEnd

GRANT USAGE ON SCHEMA public TO invoicing_lookup;
GRANT SELECT (tenant_id, status) ON electronic_documents TO invoicing_lookup;

-- RLS still applies to invoicing_lookup (FORCE): this policy lets it read
-- every row, but only through the columns granted above.
CREATE POLICY sent_lookup ON electronic_documents
    FOR SELECT TO invoicing_lookup
    USING (true);

CREATE FUNCTION tenants_with_sent_documents(p_limit int)
    RETURNS TABLE (tenant_id uuid)
    LANGUAGE sql STABLE SECURITY DEFINER
    SET search_path = pg_catalog, pg_temp
    AS $$
        SELECT DISTINCT d.tenant_id
        FROM public.electronic_documents d
        WHERE d.status = 'sent'
        ORDER BY d.tenant_id
        LIMIT p_limit
    $$;

-- A new owner needs CREATE on the schema, which 00001 took from PUBLIC; a
-- superuser skips that check, the migration role on Azure does not.
GRANT CREATE ON SCHEMA public TO invoicing_lookup;
ALTER FUNCTION tenants_with_sent_documents(int) OWNER TO invoicing_lookup;
REVOKE CREATE ON SCHEMA public FROM invoicing_lookup;
REVOKE ALL ON FUNCTION tenants_with_sent_documents(int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION tenants_with_sent_documents(int) TO app_user;

-- +goose Down
DROP FUNCTION tenants_with_sent_documents(int);
DROP POLICY sent_lookup ON electronic_documents;
REVOKE SELECT (tenant_id, status) ON electronic_documents FROM invoicing_lookup;
REVOKE USAGE ON SCHEMA public FROM invoicing_lookup;
DROP TABLE electronic_documents;
DROP TABLE numbering_resolutions;
DROP TABLE invoicing_settings;
