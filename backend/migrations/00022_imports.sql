-- Imports: the load of a tenant's initial catalog and stock from a
-- spreadsheet. The uploaded file is kept with the import (at most 10 MB,
-- checked by the API) so the background job reads it again and every
-- retry sees the same rows; there is no other file storage to operate.
-- batches_done is the job's checkpoint: it is stored in the same
-- transaction as each batch, so a retry skips the batches already written.
--
-- import_errors keeps the row errors of the validation and of the batch
-- that failed (at most 1,000 per import; errors counts them all).

-- +goose Up
CREATE TABLE imports (
    id            uuid        PRIMARY KEY,
    tenant_id     uuid        NOT NULL REFERENCES tenants (id),
    created_by    uuid        NOT NULL,
    location_id   uuid        NOT NULL,
    file_name     text        NOT NULL CHECK (length(file_name) BETWEEN 1 AND 255),
    format        text        NOT NULL CHECK (format IN ('xlsx', 'csv')),
    policy        text        NOT NULL CHECK (policy IN ('update', 'skip')),
    status        text        NOT NULL
        CHECK (status IN ('rejected', 'validated', 'queued', 'running', 'completed', 'failed')),
    total_rows    integer     NOT NULL CHECK (total_rows >= 0),
    total_products integer    NOT NULL CHECK (total_products >= 0),
    created       integer     NOT NULL DEFAULT 0 CHECK (created >= 0),
    updated       integer     NOT NULL DEFAULT 0 CHECK (updated >= 0),
    skipped       integer     NOT NULL DEFAULT 0 CHECK (skipped >= 0),
    errors        integer     NOT NULL CHECK (errors >= 0),
    batches_done  integer     NOT NULL DEFAULT 0 CHECK (batches_done >= 0),
    file          bytea       NOT NULL,
    created_at    timestamptz NOT NULL,
    started_at    timestamptz,
    finished_at   timestamptz,
    CONSTRAINT imports_tenant_id_id_key UNIQUE (tenant_id, id),
    CONSTRAINT imports_location_fkey FOREIGN KEY (tenant_id, location_id)
        REFERENCES locations (tenant_id, id)
);

CREATE TABLE import_errors (
    id         bigint  GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id  uuid    NOT NULL,
    import_id  uuid    NOT NULL,
    row_number integer NOT NULL CHECK (row_number >= 1),
    column_name text   NOT NULL,
    code       text    NOT NULL,
    message    text    NOT NULL,
    CONSTRAINT import_errors_import_fkey FOREIGN KEY (tenant_id, import_id)
        REFERENCES imports (tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX import_errors_import_idx ON import_errors (tenant_id, import_id, row_number, id);

ALTER TABLE imports ENABLE ROW LEVEL SECURITY;
ALTER TABLE imports FORCE ROW LEVEL SECURITY;
ALTER TABLE import_errors ENABLE ROW LEVEL SECURITY;
ALTER TABLE import_errors FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON imports
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY tenant_isolation ON import_errors
    USING (tenant_id = current_tenant_id())
    WITH CHECK (tenant_id = current_tenant_id());

-- +goose Down
DROP TABLE import_errors;
DROP TABLE imports;
