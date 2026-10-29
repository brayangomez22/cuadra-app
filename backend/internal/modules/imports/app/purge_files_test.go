package app_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// stored keeps an import of tenant in the fake with its file, created
// (and, unless it is still pending, finished) age before testNow.
func (e *env) storedImport(t *testing.T, tenant uuid.UUID, status domain.Status, age time.Duration) uuid.UUID {
	t.Helper()
	at := testNow.Add(-age)
	s := domain.ImportSnapshot{
		ID: uuid.Must(uuid.NewV7()), TenantID: tenant, CreatedBy: e.actor.UserID, LocationID: e.location,
		FileName: "productos.csv", Format: domain.FormatCSV, Policy: domain.UpdateExisting, Status: status,
		CreatedAt: at,
	}
	if status == domain.StatusCompleted || status == domain.StatusFailed {
		s.StartedAt, s.FinishedAt = &at, &at
	}
	imp, err := domain.RehydrateImport(s)
	require.NoError(t, err)
	require.NoError(t, e.imports.Create(t.Context(), imp, []byte("archivo"), nil))
	return imp.ID()
}

func TestPurgeFiles(t *testing.T) {
	day := 24 * time.Hour

	t.Run("borra el archivo de las importaciones terminadas hace más de 30 días, de cada tenant", func(t *testing.T) {
		e := newEnv(t)
		other := uuid.Must(uuid.NewV7())
		completed := e.storedImport(t, e.actor.TenantID, domain.StatusCompleted, 31*day)
		failed := e.storedImport(t, other, domain.StatusFailed, 40*day)
		rejected := e.storedImport(t, e.actor.TenantID, domain.StatusRejected, 35*day)

		purged, err := e.svc.PurgeFiles(t.Context())
		require.NoError(t, err)
		require.Equal(t, 3, purged)
		for _, id := range []uuid.UUID{completed, failed, rejected} {
			require.True(t, e.imports.purged(id), id)
		}
		require.Equal(t, int64(3), e.counter(t, "cuadra.imports.files_purged", "", ""))
		require.NotNil(t, e.span(t, "imports.PurgeFiles"))
	})

	t.Run("no toca las pendientes ni las recientes", func(t *testing.T) {
		e := newEnv(t)
		kept := []uuid.UUID{
			e.storedImport(t, e.actor.TenantID, domain.StatusCompleted, 29*day),
			e.storedImport(t, e.actor.TenantID, domain.StatusValidated, 90*day),
			e.storedImport(t, e.actor.TenantID, domain.StatusQueued, 90*day),
		}

		purged, err := e.svc.PurgeFiles(t.Context())
		require.NoError(t, err)
		require.Zero(t, purged)
		for _, id := range kept {
			require.False(t, e.imports.purged(id), id)
		}
	})

	t.Run("un archivo ya borrado no se vuelve a contar", func(t *testing.T) {
		e := newEnv(t)
		e.storedImport(t, e.actor.TenantID, domain.StatusCompleted, 31*day)
		_, err := e.svc.PurgeFiles(t.Context())
		require.NoError(t, err)

		purged, err := e.svc.PurgeFiles(t.Context())
		require.NoError(t, err)
		require.Zero(t, purged)
	})
}
