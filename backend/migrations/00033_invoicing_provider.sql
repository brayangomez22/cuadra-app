-- What the real technology provider needs (T36b): the company's DANE
-- municipality, the issuer of every document (P162); the tenant's account at
-- the provider, with its organization's token encrypted by the application
-- (the key is an environment variable, never in the database); and the files
-- of the accepted documents (P159). No data moves: every tenant starts
-- without municipality, without account (test mode) and without files.

-- +goose Up
ALTER TABLE tenants
    ADD COLUMN municipality_code text,
    ADD CONSTRAINT tenants_municipality_fkey FOREIGN KEY (municipality_code) REFERENCES municipalities (code);

CREATE TABLE invoicing_provider_accounts (
    tenant_id   uuid        PRIMARY KEY REFERENCES tenants (id),
    mode        text        NOT NULL CHECK (mode IN ('test', 'habilitation', 'production')),
    test_set_id text        NOT NULL DEFAULT '' CHECK (char_length(test_set_id) <= 100),
    -- AES-256-GCM of the token with its nonce first, bound to the tenant;
    -- NULL until the tenant sets it.
    token_ciphertext bytea  CHECK (octet_length(token_ciphertext) <= 8192),
    updated_at  timestamptz NOT NULL,
    CONSTRAINT invoicing_provider_accounts_test_set CHECK (mode <> 'habilitation' OR test_set_id <> '')
);

CREATE TABLE electronic_document_files (
    tenant_id   uuid        NOT NULL REFERENCES tenants (id),
    document_id uuid        NOT NULL,
    kind        text        NOT NULL CHECK (kind IN ('xml', 'pdf')),
    content     bytea       NOT NULL CHECK (octet_length(content) BETWEEN 1 AND 10485760),
    created_at  timestamptz NOT NULL,
    PRIMARY KEY (tenant_id, document_id, kind),
    CONSTRAINT electronic_document_files_document_fkey FOREIGN KEY (tenant_id, document_id)
        REFERENCES electronic_documents (tenant_id, id)
);

ALTER TABLE invoicing_provider_accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoicing_provider_accounts FORCE ROW LEVEL SECURITY;
ALTER TABLE electronic_document_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE electronic_document_files FORCE ROW LEVEL SECURITY;

-- The law keeps the documents, and their files with them: the application
-- replaces a file (the provider's same document again) but never deletes one.
REVOKE DELETE ON electronic_document_files FROM app_user;

CREATE POLICY tenant_isolation ON invoicing_provider_accounts
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());
CREATE POLICY tenant_isolation ON electronic_document_files
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- +goose Down
DROP TABLE electronic_document_files;
DROP TABLE invoicing_provider_accounts;
ALTER TABLE tenants
    DROP CONSTRAINT tenants_municipality_fkey,
    DROP COLUMN municipality_code;
