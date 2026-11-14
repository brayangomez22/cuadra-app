-- The DANE's catalog is the same for every tenant: it has no tenant_id nor
-- Row-Level Security, and the app can only read it.

-- name: ListMunicipalities :many
SELECT code, name, department_code, department_name
FROM municipalities
ORDER BY department_name, name;
