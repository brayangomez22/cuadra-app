package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// importIn builds an import in status: completed, failed, rejected,
// validated or running.
func importIn(t *testing.T, status domain.Status) *domain.Import {
	t.Helper()
	switch status {
	case domain.StatusRejected:
		return newImport(t, 3)
	case domain.StatusValidated:
		return newImport(t, 0)
	}
	imp := running(t)
	switch status {
	case domain.StatusCompleted:
		require.NoError(t, imp.Complete(testNow))
	case domain.StatusFailed:
		require.NoError(t, imp.Fail(testNow, 1))
	}
	return imp
}

func TestImportFileExpired(t *testing.T) {
	day := 24 * time.Hour
	cases := []struct {
		name    string
		status  domain.Status
		elapsed time.Duration
		want    bool
	}{
		{"una completada hace más de 30 días ya no guarda su archivo", domain.StatusCompleted, 31 * day, true},
		{"una fallida hace más de 30 días ya no guarda su archivo", domain.StatusFailed, 31 * day, true},
		{"una rechazada hace más de 30 días ya no guarda su archivo", domain.StatusRejected, 31 * day, true},
		{"una completada hace menos de 30 días conserva su archivo", domain.StatusCompleted, 29 * day, false},
		{"una validada sin confirmar conserva su archivo", domain.StatusValidated, 90 * day, false},
		{"una en curso conserva su archivo", domain.StatusRunning, 90 * day, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, importIn(t, c.status).FileExpired(testNow.Add(c.elapsed)))
		})
	}
}
