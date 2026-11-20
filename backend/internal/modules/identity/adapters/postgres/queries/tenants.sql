-- name: CreateTenant :exec
INSERT INTO tenants (id, name, nit, status, business_template, default_tax_rate, address, city, phone, demo, created_at,
                     analytics_consent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: GetTenant :one
SELECT id, name, nit, status, business_template, default_tax_rate, address, city, phone, municipality_code, demo, created_at,
       analytics_consent
FROM tenants
WHERE id = $1;

-- name: UpdateTenant :execrows
UPDATE tenants
SET name = $2, status = $3, default_tax_rate = $4, address = $5, city = $6, phone = $7,
    municipality_code = sqlc.narg(municipality_code), analytics_consent = sqlc.arg(analytics_consent)
WHERE id = $1;
