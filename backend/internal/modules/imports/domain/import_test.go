package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

var testNow = time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC)

func importParams(errors int) domain.NewImportParams {
	return domain.NewImportParams{
		TenantID:   uuid.Must(uuid.NewV7()),
		UserID:     uuid.Must(uuid.NewV7()),
		LocationID: uuid.Must(uuid.NewV7()),
		FileName:   " productos.xlsx ",
		Format:     domain.FormatXLSX,
		Policy:     domain.UpdateExisting,
		Rows:       120,
		Products:   80,
		Errors:     errors,
	}
}

func newImport(t *testing.T, errors int) *domain.Import {
	t.Helper()
	imp, err := domain.NewImport(importParams(errors), testNow)
	require.NoError(t, err)
	return imp
}

func running(t *testing.T) *domain.Import {
	t.Helper()
	imp := newImport(t, 0)
	require.NoError(t, imp.Confirm())
	require.NoError(t, imp.Start(testNow))
	return imp
}

func TestNewImport(t *testing.T) {
	t.Run("sin errores queda validada y con errores rechazada", func(t *testing.T) {
		ok := newImport(t, 0)
		require.Equal(t, domain.StatusValidated, ok.Status())
		require.Equal(t, "productos.xlsx", ok.FileName())
		require.Equal(t, 120, ok.Counts().Rows)

		rejected := newImport(t, 3)
		require.Equal(t, domain.StatusRejected, rejected.Status())
		require.Equal(t, 3, rejected.Counts().Errors)
	})

	t.Run("un archivo sin filas queda rechazado", func(t *testing.T) {
		p := importParams(0)
		p.Rows, p.Products = 0, 0
		imp, err := domain.NewImport(p, testNow)
		require.NoError(t, err)
		require.Equal(t, domain.StatusRejected, imp.Status())
	})

	t.Run("rechaza datos inválidos", func(t *testing.T) {
		tests := map[string]func(p *domain.NewImportParams){
			"sin empresa":             func(p *domain.NewImportParams) { p.TenantID = uuid.Nil },
			"sin sede":                func(p *domain.NewImportParams) { p.LocationID = uuid.Nil },
			"sin nombre de archivo":   func(p *domain.NewImportParams) { p.FileName = " " },
			"formato desconocido":     func(p *domain.NewImportParams) { p.Format = "pdf" },
			"política desconocida":    func(p *domain.NewImportParams) { p.Policy = "merge" },
			"más filas que el límite": func(p *domain.NewImportParams) { p.Rows = domain.MaxRows + 1 },
		}
		for name, change := range tests {
			t.Run(name, func(t *testing.T) {
				p := importParams(0)
				change(&p)
				_, err := domain.NewImport(p, testNow)
				require.ErrorIs(t, err, domain.ErrInvalidImport)
			})
		}
	})
}

func TestImportLifecycle(t *testing.T) {
	t.Run("solo se confirma una importación validada", func(t *testing.T) {
		imp := newImport(t, 0)
		require.NoError(t, imp.Confirm())
		require.Equal(t, domain.StatusQueued, imp.Status())
		require.ErrorIs(t, imp.Confirm(), domain.ErrImportNotValidated)

		require.ErrorIs(t, newImport(t, 2).Confirm(), domain.ErrImportNotValidated)
	})

	t.Run("empezar es idempotente para que un reintento siga", func(t *testing.T) {
		imp := running(t)
		require.NoError(t, imp.Start(testNow.Add(time.Minute)))
		require.Equal(t, domain.StatusRunning, imp.Status())
		require.Equal(t, testNow, *imp.StartedAt())
	})

	t.Run("no empieza una importación sin confirmar o terminada", func(t *testing.T) {
		require.ErrorIs(t, newImport(t, 0).Start(testNow), domain.ErrImportNotQueued)

		done := running(t)
		require.NoError(t, done.Complete(testNow))
		require.ErrorIs(t, done.Start(testNow), domain.ErrImportFinished)
	})

	t.Run("cada lote suma sus conteos y avanza el punto de control", func(t *testing.T) {
		imp := running(t)
		require.NoError(t, imp.RecordBatch(domain.BatchResult{Created: 3, Updated: 1}))
		require.NoError(t, imp.RecordBatch(domain.BatchResult{Created: 2, Skipped: 4}))

		require.Equal(t, 2, imp.BatchesDone())
		c := imp.Counts()
		require.Equal(t, []int{5, 1, 4}, []int{c.Created, c.Updated, c.Skipped})
	})

	t.Run("un lote solo se registra mientras corre", func(t *testing.T) {
		require.ErrorIs(t, newImport(t, 0).RecordBatch(domain.BatchResult{Created: 1}), domain.ErrImportNotRunning)
	})

	t.Run("termina completada o fallida con sus errores", func(t *testing.T) {
		done := running(t)
		require.NoError(t, done.Complete(testNow.Add(time.Hour)))
		require.Equal(t, domain.StatusCompleted, done.Status())
		require.Equal(t, testNow.Add(time.Hour), *done.FinishedAt())

		failed := running(t)
		require.NoError(t, failed.Fail(testNow, 2))
		require.Equal(t, domain.StatusFailed, failed.Status())
		require.Equal(t, 2, failed.Counts().Errors)
		require.ErrorIs(t, failed.Complete(testNow), domain.ErrImportNotRunning)
	})
}
