//go:build integration

package migrations_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// The migrations of T38: 00034_audit_events and 00035_refresh_token_family,
// which stores when each session (refresh token family) started.
const (
	auditEventsAdded         = 34
	refreshTokenFamilyStarts = 35
)

func TestRefreshTokenFamilyMigration(t *testing.T) {
	t.Run("cada token queda con el inicio de su familia: el token más viejo de ella", func(t *testing.T) {
		s := dbtest.StartAt(t, auditEventsAdded)
		tenant := seedTenant(t, s, "generica", false)
		conn := connect(t, s.SuperuserURL)
		var user uuid.UUID
		require.NoError(t, conn.QueryRow(t.Context(), "SELECT id FROM users WHERE tenant_id = $1", tenant).Scan(&user))
		start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
		family, other := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
		insert := func(family uuid.UUID, created time.Time) {
			_, err := conn.Exec(t.Context(), `
				INSERT INTO refresh_tokens (id, tenant_id, user_id, family_id, token_hash, expires_at, created_at)
				VALUES ($1, $2, $3, $4, gen_random_bytes(32), $5, $5)`,
				uuid.Must(uuid.NewV7()), tenant, user, family, created)
			require.NoError(t, err)
		}
		insert(family, start)
		insert(family, start.Add(48*time.Hour))
		insert(other, start.Add(time.Hour))

		s.MigrateTo(t, refreshTokenFamilyStarts)

		rows, err := conn.Query(t.Context(), "SELECT family_id, family_started_at FROM refresh_tokens")
		require.NoError(t, err)
		got := map[uuid.UUID][]time.Time{}
		for rows.Next() {
			var f uuid.UUID
			var started time.Time
			require.NoError(t, rows.Scan(&f, &started))
			got[f] = append(got[f], started.UTC())
		}
		require.NoError(t, rows.Err())
		require.Equal(t, []time.Time{start, start}, got[family])
		require.Equal(t, []time.Time{start.Add(time.Hour)}, got[other])
	})

	t.Run("el Down quita la columna", func(t *testing.T) {
		s := dbtest.StartAt(t, refreshTokenFamilyStarts)
		s.MigrateDownTo(t, auditEventsAdded)
		var exists bool
		require.NoError(t, connect(t, s.SuperuserURL).QueryRow(t.Context(), `
			SELECT EXISTS (SELECT 1 FROM information_schema.columns
			               WHERE table_name = 'refresh_tokens' AND column_name = 'family_started_at')`).Scan(&exists))
		require.False(t, exists)
	})
}
