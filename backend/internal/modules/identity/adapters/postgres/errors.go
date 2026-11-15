package postgres

// usersEmailKey is the unique (tenant_id, email) constraint of users.
const usersEmailKey = "users_tenant_id_email_key"

// tenantsMunicipalityKey is the foreign key of a tenant's municipality to
// the DANE catalog.
const tenantsMunicipalityKey = "tenants_municipality_fkey"
