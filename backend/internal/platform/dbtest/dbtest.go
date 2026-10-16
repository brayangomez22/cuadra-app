// Package dbtest starts PostgreSQL for integration tests, with the migrations
// applied, and connects as app_user: the same role and permissions as the API.
// The migrations run as an owner that is not a superuser nor BYPASSRLS, like
// the administrator of Azure Database for PostgreSQL, so a migration that
// only works as a superuser fails here first.
package dbtest

import (
	"context"
	"math"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/migrations"
)

const (
	image = "postgres:16-alpine"
	// Throwaway credentials for a disposable container.
	superUser   = "cuadra"
	superPass   = "cuadra"
	ownerUser   = "cuadra_owner"
	ownerPass   = "cuadra_owner"
	appUser     = "app_user"
	appPassword = "app_user"
)

// createOwner makes the schema owner: it may create roles and databases but
// is not a superuser nor BYPASSRLS, as on Azure, and owns the database (and
// so the public schema).
const createOwner = `
CREATE ROLE cuadra_owner LOGIN PASSWORD 'cuadra_owner' NOSUPERUSER NOBYPASSRLS CREATEDB CREATEROLE;
ALTER DATABASE cuadra OWNER TO cuadra_owner;`

// Server is a running PostgreSQL with the migrations applied.
type Server struct {
	// AdminURL connects as the owner of the schema, the role that runs the
	// migrations. It is not a superuser.
	AdminURL string
	// SuperuserURL connects as the container's superuser, which bypasses
	// Row-Level Security: only for tests that inspect every tenant's rows or
	// set up what the owner cannot.
	SuperuserURL string
	// AppURL connects as app_user, subject to Row-Level Security.
	AppURL string

	container *postgres.PostgresContainer
}

// New returns a DB connected as app_user to the PostgreSQL shared by the test
// package. Tests sharing it must use their own tenant ids.
func New(t testing.TB) *db.DB {
	t.Helper()
	return Open(t, Shared(t).AppURL)
}

// Open connects to url without telemetry and closes the pool when t ends.
func Open(t testing.TB, url string) *db.DB {
	t.Helper()
	d, err := db.Open(t.Context(), url, tracenoop.NewTracerProvider(), metricnoop.NewMeterProvider())
	require.NoError(t, err)
	t.Cleanup(d.Close)
	return d
}

var (
	sharedOnce   sync.Once
	sharedServer *Server
	errShared    error
)

// Shared returns the PostgreSQL shared by the test package, starting it on
// first use. It is removed when the test binary exits.
func Shared(t testing.TB) *Server {
	t.Helper()
	sharedOnce.Do(func() {
		// Not tied to t: the server outlives the test that started it.
		sharedServer, errShared = start(context.Background(), math.MaxInt64)
	})
	require.NoError(t, errShared)
	return sharedServer
}

// Start starts a PostgreSQL dedicated to t, for tests that need to control the
// server (stop it, roll migrations back). It is removed when t ends.
func Start(t testing.TB) *Server { return StartAt(t, math.MaxInt64) }

// StartAt is Start with the migrations applied only up to version, for
// tests that load data in an older schema and then migrate it (MigrateTo).
func StartAt(t testing.TB, version int64) *Server {
	t.Helper()
	s, err := start(t.Context(), version)
	if s != nil {
		t.Cleanup(func() { _ = s.container.Terminate(context.WithoutCancel(t.Context())) })
	}
	require.NoError(t, err)
	return s
}

// Stop stops the server, leaving the URLs unreachable.
func (s *Server) Stop(t testing.TB) {
	t.Helper()
	timeout := 10 * time.Second
	require.NoError(t, s.container.Stop(t.Context(), &timeout))
}

// MigrateTo applies the pending migrations up to version as the owner.
func (s *Server) MigrateTo(t testing.TB, version int64) {
	t.Helper()
	require.NoError(t, s.TryMigrateTo(t, version))
}

// TryMigrateTo is MigrateTo returning the error, for tests that expect a
// migration to fail.
func (s *Server) TryMigrateTo(t testing.TB, version int64) error {
	t.Helper()
	_, err := s.provider(t).UpTo(t.Context(), version)
	return err
}

// MigrateDownTo rolls the migrations back down to version as the owner.
func (s *Server) MigrateDownTo(t testing.TB, version int64) {
	t.Helper()
	_, err := s.provider(t).DownTo(t.Context(), version)
	require.NoError(t, err)
}

func (s *Server) provider(t testing.TB) *goose.Provider {
	t.Helper()
	cfg, err := pgx.ParseConfig(s.AdminURL)
	require.NoError(t, err)
	sqlDB := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { _ = sqlDB.Close() })
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	require.NoError(t, err)
	return p
}

func start(ctx context.Context, version int64) (*Server, error) {
	container, err := postgres.Run(ctx, image,
		postgres.WithDatabase("cuadra"),
		postgres.WithUsername(superUser),
		postgres.WithPassword(superPass),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, err
	}
	s := &Server{container: container}

	s.SuperuserURL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return s, err
	}
	if err := exec(ctx, s.SuperuserURL, createOwner); err != nil {
		return s, err
	}
	if s.AdminURL, err = withUser(s.SuperuserURL, ownerUser, ownerPass); err != nil {
		return s, err
	}
	if s.AppURL, err = withUser(s.SuperuserURL, appUser, appPassword); err != nil {
		return s, err
	}
	return s, db.MigrateTo(ctx, s.AdminURL, appPassword, version)
}

// exec runs sql on a short-lived connection to url.
func exec(ctx context.Context, url, sql string) error {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()
	_, err = conn.Exec(ctx, sql)
	return err
}

// withUser returns rawURL with other credentials.
func withUser(rawURL, user, password string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	u.User = url.UserPassword(user, password)
	return u.String(), nil
}
