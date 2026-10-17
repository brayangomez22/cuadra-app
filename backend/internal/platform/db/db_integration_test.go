//go:build integration

package db_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx/healthapi"
	"github.com/brayangomez22/cuadra-app/backend/migrations"
)

// singleConn opens a pool with one connection, so consecutive transactions are
// guaranteed to run on the same session.
func singleConn(t *testing.T) *db.DB {
	t.Helper()
	return dbtest.Open(t, dbtest.Shared(t).AppURL+"&pool_max_conns=1")
}

func currentTenant(ctx context.Context, tx pgx.Tx) (string, error) {
	var tenant string
	err := tx.QueryRow(ctx, "SELECT coalesce(current_setting('app.tenant_id', true), '')").Scan(&tenant)
	return tenant, err
}

func TestWithTenantTx(t *testing.T) {
	t.Run("WithTenantTx deja app.tenant_id visible dentro de la transacción", func(t *testing.T) {
		d := dbtest.New(t)
		tenantID := uuid.Must(uuid.NewV7())

		var got string
		err := d.WithTenantTx(t.Context(), tenantID, func(tx pgx.Tx) error {
			var err error
			got, err = currentTenant(t.Context(), tx)
			return err
		})

		require.NoError(t, err)
		require.Equal(t, tenantID.String(), got)
	})

	t.Run("app.tenant_id no queda visible fuera de la transacción", func(t *testing.T) {
		d := singleConn(t)
		require.NoError(t, d.WithTenantTx(t.Context(), uuid.Must(uuid.NewV7()), func(pgx.Tx) error { return nil }))

		var got string
		err := d.WithTx(t.Context(), func(tx pgx.Tx) error {
			var err error
			got, err = currentTenant(t.Context(), tx)
			return err
		})

		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("app.tenant_id no se filtra a la siguiente transacción tras un error", func(t *testing.T) {
		d := singleConn(t)
		boom := errors.New("boom")
		err := d.WithTenantTx(t.Context(), uuid.Must(uuid.NewV7()), func(pgx.Tx) error { return boom })
		require.ErrorIs(t, err, boom)

		var got string
		err = d.WithTx(t.Context(), func(tx pgx.Tx) error {
			var err error
			got, err = currentTenant(t.Context(), tx)
			return err
		})

		require.NoError(t, err)
		require.Empty(t, got)
	})
}

func TestWithTx(t *testing.T) {
	// A temp table lives in the session, which is why these tests use one connection.
	setup := func(t *testing.T) *db.DB {
		t.Helper()
		d := singleConn(t)
		require.NoError(t, d.WithTx(t.Context(), func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), "CREATE TEMP TABLE probe (n int)")
			return err
		}))
		return d
	}
	insert := func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), "INSERT INTO probe VALUES (1)")
		return err
	}
	count := func(t *testing.T, d *db.DB) int {
		t.Helper()
		var n int
		require.NoError(t, d.WithTx(t.Context(), func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(), "SELECT count(*) FROM probe").Scan(&n)
		}))
		return n
	}

	t.Run("WithTx confirma los cambios cuando fn termina sin error", func(t *testing.T) {
		d := setup(t)

		require.NoError(t, d.WithTx(t.Context(), insert))

		require.Equal(t, 1, count(t, d))
	})

	t.Run("un error dentro de fn hace rollback y se devuelve", func(t *testing.T) {
		d := setup(t)
		boom := errors.New("boom")

		err := d.WithTx(t.Context(), func(tx pgx.Tx) error {
			require.NoError(t, insert(tx))
			return boom
		})

		require.ErrorIs(t, err, boom)
		require.Equal(t, 0, count(t, d))
	})

	t.Run("un panic dentro de fn hace rollback y se propaga", func(t *testing.T) {
		d := setup(t)

		require.PanicsWithValue(t, "boom", func() {
			_ = d.WithTx(t.Context(), func(tx pgx.Tx) error {
				require.NoError(t, insert(tx))
				panic("boom")
			})
		})

		require.Equal(t, 0, count(t, d))
	})
}

func TestAppRole(t *testing.T) {
	t.Run("la app se conecta como app_user sin superusuario ni BYPASSRLS", func(t *testing.T) {
		d := dbtest.New(t)

		var user string
		var super, bypass bool
		err := d.WithTx(t.Context(), func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(),
				"SELECT rolname, rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user",
			).Scan(&user, &super, &bypass)
		})

		require.NoError(t, err)
		require.Equal(t, "app_user", user)
		require.False(t, super)
		require.False(t, bypass)
	})

	t.Run("app_user no puede crear tablas en public", func(t *testing.T) {
		d := dbtest.New(t)

		err := d.WithTx(t.Context(), func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), "CREATE TABLE public.intruder (id int)")
			return err
		})

		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		require.Equal(t, "42501", pgErr.Code, "insufficient_privilege")
	})

	t.Run("Open rechaza conectarse como superusuario", func(t *testing.T) {
		d, err := db.Open(t.Context(), dbtest.Shared(t).SuperuserURL, tracenoop.NewTracerProvider(), metricnoop.NewMeterProvider())

		require.ErrorIs(t, err, db.ErrUnsafeRole)
		require.Nil(t, d)
	})

	t.Run("Open rechaza conectarse como el dueño de las tablas", func(t *testing.T) {
		d, err := db.Open(t.Context(), dbtest.Shared(t).AdminURL, tracenoop.NewTracerProvider(), metricnoop.NewMeterProvider())

		require.ErrorIs(t, err, db.ErrUnsafeRole)
		require.Nil(t, d)
	})
}

func TestMigrations(t *testing.T) {
	t.Run("el dueño que aplica las migraciones no es superusuario ni BYPASSRLS, como en Azure", func(t *testing.T) {
		sqlDB := stdlib.OpenDB(*mustParseConfig(t, dbtest.Shared(t).AdminURL))
		t.Cleanup(func() { _ = sqlDB.Close() })

		var super, bypass, ownsProducts bool
		require.NoError(t, sqlDB.QueryRowContext(t.Context(), `
			SELECT r.rolsuper, r.rolbypassrls, (SELECT tableowner = current_user FROM pg_tables WHERE tablename = 'products')
			FROM pg_roles r WHERE r.rolname = current_user`).Scan(&super, &bypass, &ownsProducts))
		require.False(t, super)
		require.False(t, bypass)
		require.True(t, ownsProducts)
	})

	t.Run("las migraciones se revierten y se vuelven a aplicar", func(t *testing.T) {
		s := dbtest.Start(t)
		sqlDB := stdlib.OpenDB(*mustParseConfig(t, s.AdminURL))
		t.Cleanup(func() { _ = sqlDB.Close() })
		provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
		require.NoError(t, err)

		_, err = provider.DownTo(t.Context(), 0)
		require.NoError(t, err)
		require.NoError(t, db.Migrate(t.Context(), s.AdminURL, "app_user"))

		dbtest.Open(t, s.AppURL)
	})
}

func mustParseConfig(t *testing.T, url string) *pgx.ConnConfig {
	t.Helper()
	cfg, err := pgx.ParseConfig(url)
	require.NoError(t, err)
	return cfg
}

func TestReadyzWithDatabase(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	readyz := func(d *db.DB) int {
		rec := httptest.NewRecorder()
		health := healthapi.Handler(healthapi.NewStrictHandler(httpx.NewHealth(log, d), nil))
		health.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		return rec.Code
	}

	t.Run("/readyz responde 200 con la BD disponible", func(t *testing.T) {
		require.Equal(t, http.StatusOK, readyz(dbtest.New(t)))
	})

	t.Run("/readyz responde 503 si la BD no está disponible", func(t *testing.T) {
		s := dbtest.Start(t)
		d := dbtest.Open(t, s.AppURL)
		require.Equal(t, http.StatusOK, readyz(d))

		s.Stop(t)

		require.Equal(t, http.StatusServiceUnavailable, readyz(d))
	})
}

func TestQuerySpans(t *testing.T) {
	t.Run("nombra el span de la query con su nombre de sqlc", func(t *testing.T) {
		recorder := tracetest.NewSpanRecorder()
		tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
		t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
		d, err := db.Open(t.Context(), dbtest.Shared(t).AppURL, tp, metricnoop.NewMeterProvider())
		require.NoError(t, err)
		t.Cleanup(d.Close)
		ctx, parent := tp.Tracer("test").Start(t.Context(), "parent")

		err = d.WithTx(ctx, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "-- name: PingQuery :exec\nSELECT 1")
			return err
		})
		parent.End()
		require.NoError(t, err)

		var names []string
		for _, s := range recorder.Ended() {
			names = append(names, s.Name())
		}
		require.Contains(t, names, "PingQuery :exec")
	})
}

// insertTenant stores the tenant row of id within tx: tenants is a business
// table whose policy compares id with app.tenant_id.
func insertTenant(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, "INSERT INTO tenants (id, name, nit, status, business_template, created_at) VALUES ($1, 'Ferretería', '890903938-8', 'active', 'ferreteria', now())", id)
	return err
}

// tenantExists reports whether the tenant row of id was committed.
func tenantExists(t *testing.T, d *db.DB, id uuid.UUID) bool {
	t.Helper()
	var n int
	require.NoError(t, d.WithTenantTx(t.Context(), id, func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(), "SELECT count(*) FROM tenants WHERE id = $1", id).Scan(&n)
	}))
	return n == 1
}

func TestInTenantTx(t *testing.T) {
	t.Run("WithTenantTx con una transacción en el contexto se une a ella y su rollback deshace todo", func(t *testing.T) {
		d := dbtest.New(t)
		tenantID := uuid.Must(uuid.NewV7())
		boom := errors.New("boom")

		err := d.InTenantTx(t.Context(), tenantID, func(ctx context.Context, _ pgx.Tx) error {
			require.NoError(t, d.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error { return insertTenant(ctx, tx, tenantID) }))
			// La escritura de la transacción interna se ve en la externa.
			require.NoError(t, d.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
				var n int
				if err := tx.QueryRow(ctx, "SELECT count(*) FROM tenants WHERE id = $1", tenantID).Scan(&n); err != nil {
					return err
				}
				require.Equal(t, 1, n)
				return nil
			}))
			return boom
		})

		require.ErrorIs(t, err, boom)
		require.False(t, tenantExists(t, d, tenantID))
	})

	t.Run("confirma lo escrito por la transacción unida", func(t *testing.T) {
		d := dbtest.New(t)
		tenantID := uuid.Must(uuid.NewV7())

		err := d.InTenantTx(t.Context(), tenantID, func(ctx context.Context, _ pgx.Tx) error {
			return d.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error { return insertTenant(ctx, tx, tenantID) })
		})

		require.NoError(t, err)
		require.True(t, tenantExists(t, d, tenantID))
	})

	t.Run("un error de la transacción unida deshace solo lo suyo", func(t *testing.T) {
		d := dbtest.New(t)
		tenantID := uuid.Must(uuid.NewV7())
		boom := errors.New("boom")

		err := d.InTenantTx(t.Context(), tenantID, func(ctx context.Context, tx pgx.Tx) error {
			if err := insertTenant(ctx, tx, tenantID); err != nil {
				return err
			}
			inner := d.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
				if _, err := tx.Exec(ctx, "UPDATE tenants SET name = 'Otro' WHERE id = $1", tenantID); err != nil {
					return err
				}
				return boom
			})
			require.ErrorIs(t, inner, boom)
			return nil
		})

		require.NoError(t, err)
		var name string
		require.NoError(t, d.WithTenantTx(t.Context(), tenantID, func(tx pgx.Tx) error {
			return tx.QueryRow(t.Context(), "SELECT name FROM tenants WHERE id = $1", tenantID).Scan(&name)
		}))
		require.Equal(t, "Ferretería", name)
	})

	t.Run("rechaza unirse con otro tenant", func(t *testing.T) {
		d := dbtest.New(t)
		tenantA, tenantB := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

		err := d.InTenantTx(t.Context(), tenantA, func(ctx context.Context, _ pgx.Tx) error {
			return d.WithTenantTx(ctx, tenantB, func(pgx.Tx) error {
				t.Fatal("no debe correr con el tenant equivocado")
				return nil
			})
		})

		require.ErrorIs(t, err, db.ErrTenantMismatch)
	})

	t.Run("InTenantTx anidado se une a la transacción del contexto", func(t *testing.T) {
		d := dbtest.New(t)
		tenantID := uuid.Must(uuid.NewV7())
		boom := errors.New("boom")

		err := d.InTenantTx(t.Context(), tenantID, func(ctx context.Context, _ pgx.Tx) error {
			require.NoError(t, d.InTenantTx(ctx, tenantID, func(ctx context.Context, tx pgx.Tx) error {
				return insertTenant(ctx, tx, tenantID)
			}))
			return boom
		})

		require.ErrorIs(t, err, boom)
		require.False(t, tenantExists(t, d, tenantID))
	})

	t.Run("rechaza un tenant vacío", func(t *testing.T) {
		d := dbtest.New(t)
		err := d.InTenantTx(t.Context(), uuid.Nil, func(context.Context, pgx.Tx) error { return nil })
		require.ErrorIs(t, err, db.ErrNoTenant)
	})
}
