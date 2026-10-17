-- The IVA rate each tenant proposes to its new products, seeded at sign-up
-- from its business template (default_tax_rate in the template's YAML).
--
-- Every existing tenant has the ferreteria template, whose rate is 0.19.
-- ADD COLUMN with a DEFAULT writes the value in every row without an UPDATE,
-- so row-level security does not filter it and there is no backfill to
-- count. The default stays as the general IVA rate for whoever inserts a
-- tenant directly (tests, scripts); the sign-up always sends its template's.

-- +goose Up
ALTER TABLE tenants ADD COLUMN default_tax_rate NUMERIC(5,4) NOT NULL DEFAULT 0.19
    CHECK (default_tax_rate >= 0 AND default_tax_rate < 1);

-- +goose Down
ALTER TABLE tenants DROP COLUMN default_tax_rate;
