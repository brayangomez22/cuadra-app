//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

var now = time.Date(2026, 11, 19, 15, 0, 0, 0, time.UTC)

// seed stores a tenant and one of its users directly: onboarding does not
// depend on identity, but its table references both.
func seed(t *testing.T, d *db.DB, name, template string) (tenant, user uuid.UUID) {
	t.Helper()
	tenant, user = uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	require.NoError(t, d.WithTenantTx(t.Context(), tenant, func(tx pgx.Tx) error {
		if _, err := tx.Exec(t.Context(),
			"INSERT INTO tenants (id, name, nit, status, business_template, created_at) VALUES ($1, $2, '900123456-8', 'active', $3, $4)",
			tenant, name, template, now); err != nil {
			return err
		}
		_, err := tx.Exec(t.Context(),
			"INSERT INTO users (id, tenant_id, email, name, password_hash, role, active, created_at) VALUES ($1, $2, 'ana@negocio.co', 'Ana', 'x', 'owner', true, $3)",
			user, tenant, now)
		return err
	}))
	return tenant, user
}
