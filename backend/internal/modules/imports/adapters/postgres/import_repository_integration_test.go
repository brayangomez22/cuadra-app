//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	jobsadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/jobs"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/jobs"
)

// now has no sub-microsecond part, so it survives a round trip through timestamptz.
var now = time.Date(2026, 10, 26, 20, 30, 15, 123456000, time.UTC)

// Compile-time checks: the adapters satisfy the ports.
var (
	_ domain.ImportRepository = (*postgres.ImportRepository)(nil)
	_ app.Transactor          = postgres.Transactor{}
)

type fixture struct {
	db      *db.DB
	imports *postgres.ImportRepository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	d := dbtest.New(t)
	client, err := jobs.New(jobs.Config{DB: d})
	require.NoError(t, err)
	return fixture{db: d, imports: postgres.NewImportRepository(d, jobsadapter.Enqueuer(client))}
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// tenant stores a tenant and a location directly: imports do not depend on
// the identity and inventory modules, but its tables reference theirs.
func (f fixture) tenant(t *testing.T) (tenantID, locationID uuid.UUID) {
	t.Helper()
	tenantID, locationID = newID(), newID()
	require.NoError(t, f.db.WithTenantTx(t.Context(), tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(t.Context(), `INSERT INTO tenants (id, name, nit, status, business_template, created_at)
			VALUES ($1, 'Almacén', '900123456-8', 'active', 'ropa_calzado', $2)`, tenantID, now); err != nil {
			return err
		}
		_, err := tx.Exec(t.Context(), `INSERT INTO locations (id, tenant_id, name, active, created_at)
			VALUES ($1, $2, 'Principal', true, $3)`, locationID, tenantID, now)
		return err
	}))
	return tenantID, locationID
}

// stored creates a validated import (or a rejected one with errs).
func (f fixture) stored(t *testing.T, tenantID, locationID uuid.UUID, errs ...domain.RowError) *domain.Import {
	t.Helper()
	imp, err := domain.NewImport(domain.NewImportParams{
		TenantID: tenantID, UserID: newID(), LocationID: locationID, FileName: "productos.xlsx",
		Format: domain.FormatXLSX, Policy: domain.UpdateExisting, Rows: 3, Products: 2, Errors: len(errs),
	}, now)
	require.NoError(t, err)
	require.NoError(t, f.imports.Create(t.Context(), imp, []byte("contenido"), errs))
	return imp
}

// jobsOf counts the jobs enqueued for an import (River's tables have no RLS).
func (f fixture) jobsOf(t *testing.T, importID uuid.UUID) int {
	t.Helper()
	var n int
	require.NoError(t, f.db.Pool().QueryRow(t.Context(),
		"SELECT count(*) FROM river_job WHERE kind = 'imports.run' AND args->>'import_id' = $1", importID.String()).Scan(&n))
	return n
}

func TestImportRepository(t *testing.T) {
	f := newFixture(t)
	tenant, location := f.tenant(t)

	t.Run("guarda una importación con su archivo y sus errores, y la lee", func(t *testing.T) {
		errs := []domain.RowError{
			{Row: 9, Column: "precio", Code: domain.CodeNotANumber},
			{Row: 3, Code: "invalid_sku", Message: "El SKU no es válido."},
		}
		imp := f.stored(t, tenant, location, errs...)

		got, err := f.imports.Get(t.Context(), tenant, imp.ID())
		require.NoError(t, err)
		require.Equal(t, imp.Snapshot(), got.Snapshot())
		file, err := f.imports.File(t.Context(), tenant, imp.ID())
		require.NoError(t, err)
		require.Equal(t, []byte("contenido"), file)
		page, total, err := f.imports.Errors(t.Context(), tenant, imp.ID(), 1, 0)
		require.NoError(t, err)
		require.Equal(t, 2, total)
		require.Equal(t, []domain.RowError{errs[1]}, page, "by row")
	})

	t.Run("confirmar guarda el estado y encola su trabajo en la misma transacción", func(t *testing.T) {
		imp := f.stored(t, tenant, location)
		require.NoError(t, imp.Confirm())

		require.NoError(t, f.imports.Queue(t.Context(), imp))

		got, err := f.imports.Get(t.Context(), tenant, imp.ID())
		require.NoError(t, err)
		require.Equal(t, domain.StatusQueued, got.Status())
		require.Equal(t, 1, f.jobsOf(t, imp.ID()))
	})

	t.Run("si el trabajo no se puede encolar, la importación no queda confirmada", func(t *testing.T) {
		failing := postgres.NewImportRepository(f.db, func(context.Context, pgx.Tx, uuid.UUID, uuid.UUID) error {
			return errors.New("queue down")
		})
		imp := f.stored(t, tenant, location)
		require.NoError(t, imp.Confirm())

		require.Error(t, failing.Queue(t.Context(), imp))

		got, err := f.imports.Get(t.Context(), tenant, imp.ID())
		require.NoError(t, err)
		require.Equal(t, domain.StatusValidated, got.Status())
	})

	t.Run("el punto de control de un lote se deshace con su transacción", func(t *testing.T) {
		imp := f.stored(t, tenant, location)
		require.NoError(t, imp.Confirm())
		require.NoError(t, imp.Start(now))
		require.NoError(t, f.imports.Save(t.Context(), imp, nil))
		tx := postgres.NewTransactor(f.db)

		err := tx.InTx(t.Context(), tenant, func(ctx context.Context) error {
			require.NoError(t, imp.RecordBatch(domain.BatchResult{Created: 5}))
			require.NoError(t, f.imports.Save(ctx, imp, nil))
			return errors.New("the batch failed")
		})
		require.Error(t, err)

		got, err := f.imports.Get(t.Context(), tenant, imp.ID())
		require.NoError(t, err)
		require.Equal(t, 0, got.BatchesDone())
		require.Equal(t, 0, got.Counts().Created)
	})

	t.Run("aislamiento: otra empresa no ve, no lee ni guarda la importación del A", func(t *testing.T) {
		imp := f.stored(t, tenant, location)
		other, _ := f.tenant(t)

		_, err := f.imports.Get(t.Context(), other, imp.ID())
		require.ErrorIs(t, err, domain.ErrImportNotFound)
		_, err = f.imports.File(t.Context(), other, imp.ID())
		require.ErrorIs(t, err, domain.ErrImportNotFound)
		_, _, err = f.imports.Errors(t.Context(), other, imp.ID(), 10, 0)
		require.ErrorIs(t, err, domain.ErrImportNotFound)

		snapshot := imp.Snapshot()
		snapshot.TenantID = other
		foreign, err := domain.RehydrateImport(snapshot)
		require.NoError(t, err)
		require.NoError(t, foreign.Confirm())
		require.ErrorIs(t, f.imports.Save(t.Context(), foreign, nil), domain.ErrImportNotFound)
		require.Equal(t, 0, f.jobsOf(t, imp.ID()))
	})
}
