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
func New(tb testing.TB) *db.DB {
	tb.Helper()
	return Open(tb, Shared(tb).AppURL)
}

// Open connects to url without telemetry and closes the pool when t ends.
func Open(tb testing.TB, url string) *db.DB {
	tb.Helper()
	d, err := db.Open(tb.Context(), url, tracenoop.NewTracerProvider(), metricnoop.NewMeterProvider())
	require.NoError(tb, err)
	tb.Cleanup(d.Close)
	return d
}

var (
	sharedOnce   sync.Once
	sharedServer *Server
	errShared    error
)

// Shared returns the PostgreSQL shared by the test package, starting it on
// first use. It is removed when the test binary exits.
func Shared(tb testing.TB) *Server {
	tb.Helper()
	sharedOnce.Do(func() {
		// Not tied to t: the server outlives the test that started it.
		sharedServer, errShared = start(context.Background(), math.MaxInt64)
	})
	require.NoError(tb, errShared)
	return sharedServer
}

// Start starts a PostgreSQL dedicated to t, for tests that need to control the
// server (stop it, roll migrations back). It is removed when t ends.
func Start(tb testing.TB) *Server {
	tb.Helper()
	return StartAt(tb, math.MaxInt64)
}

// StartAt is Start with the migrations applied only up to version, for
// tests that load data in an older schema and then migrate it (MigrateTo).
func StartAt(tb testing.TB, version int64) *Server {
	tb.Helper()
	s, err := start(tb.Context(), version)
	if s != nil {
		tb.Cleanup(func() { _ = s.container.Terminate(context.WithoutCancel(tb.Context())) })
	}
	require.NoError(tb, err)
	return s
}

// Stop stops the server, leaving the URLs unreachable.
func (s *Server) Stop(tb testing.TB) {
	tb.Helper()
	timeout := 10 * time.Second
	require.NoError(tb, s.container.Stop(tb.Context(), &timeout))
}

// MigrateTo applies the pending migrations up to version as the owner.
func (s *Server) MigrateTo(tb testing.TB, version int64) {
	tb.Helper()
	require.NoError(tb, s.TryMigrateTo(tb, version))
}

// TryMigrateTo is MigrateTo returning the error, for tests that expect a
// migration to fail.
func (s *Server) TryMigrateTo(tb testing.TB, version int64) error {
	tb.Helper()
	_, err := s.provider(tb).UpTo(tb.Context(), version)
	return err
}

// MigrateDownTo rolls the migrations back down to version as the owner.
func (s *Server) MigrateDownTo(tb testing.TB, version int64) {
	tb.Helper()
	require.NoError(tb, s.TryMigrateDownTo(tb, version))
}

// TryMigrateDownTo is MigrateDownTo returning the error, for tests that
// expect a Down to fail.
func (s *Server) TryMigrateDownTo(tb testing.TB, version int64) error {
	tb.Helper()
	_, err := s.provider(tb).DownTo(tb.Context(), version)
	return err
}

func (s *Server) provider(tb testing.TB) *goose.Provider {
	tb.Helper()
	cfg, err := pgx.ParseConfig(s.AdminURL)
	require.NoError(tb, err)
	sqlDB := stdlib.OpenDB(*cfg)
	tb.Cleanup(func() { _ = sqlDB.Close() })
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	require.NoError(tb, err)
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
