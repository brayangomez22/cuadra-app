// Package dbtest starts PostgreSQL for integration tests, with the migrations
// applied, and connects as app_user: the same role and permissions as the API.
package dbtest

import (
	"context"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

const (
	image = "postgres:16-alpine"
	// Throwaway credentials for a disposable container.
	adminUser   = "cuadra"
	adminPass   = "cuadra"
	appUser     = "app_user"
	appPassword = "app_user"
)

// Server is a running PostgreSQL with the migrations applied.
type Server struct {
	// AdminURL connects as the superuser that owns the schema.
	AdminURL string
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
		sharedServer, errShared = start(context.Background())
	})
	require.NoError(t, errShared)
	return sharedServer
}

// Start starts a PostgreSQL dedicated to t, for tests that need to control the
// server (stop it, roll migrations back). It is removed when t ends.
func Start(t testing.TB) *Server {
	t.Helper()
	s, err := start(t.Context())
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

func start(ctx context.Context) (*Server, error) {
	container, err := postgres.Run(ctx, image,
		postgres.WithDatabase("cuadra"),
		postgres.WithUsername(adminUser),
		postgres.WithPassword(adminPass),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, err
	}
	s := &Server{container: container}

	s.AdminURL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return s, err
	}
	u, err := url.Parse(s.AdminURL)
	if err != nil {
		return s, err
	}
	u.User = url.UserPassword(appUser, appPassword)
	s.AppURL = u.String()

	return s, db.Migrate(ctx, s.AdminURL, appPassword)
}
