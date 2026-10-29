package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// env is a Service wired to fakes, with recorders for spans and metrics.
type env struct {
	svc          *app.Service
	clock        *fakeClock
	tenants      *fakeTenants
	users        *fakeUsers
	tokens       *fakeTokens
	demo         *fakeDemo
	hasher       *fakeHasher
	setup        *fakeSetup
	modules      *fakeModules
	ipLimiter    *fakeLimiter
	emailLimiter *fakeLimiter
	demoLimiter  *fakeLimiter
	spans        *tracetest.SpanRecorder
	metrics      *sdkmetric.ManualReader
}

// newEnv builds the env with the demo on; opts change the service's
// dependencies before it is built.
func newEnv(t *testing.T, opts ...func(*app.Deps)) *env {
	t.Helper()
	clock := &fakeClock{now: time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)}
	users := &fakeUsers{byID: map[uuid.UUID]*domain.User{}}
	e := &env{
		clock:        clock,
		users:        users,
		tenants:      &fakeTenants{byID: map[uuid.UUID]*domain.Tenant{}, users: users},
		tokens:       &fakeTokens{byID: map[uuid.UUID]*domain.RefreshToken{}},
		hasher:       &fakeHasher{},
		setup:        &fakeSetup{},
		modules:      &fakeModules{active: map[uuid.UUID][]string{}},
		ipLimiter:    &fakeLimiter{denied: map[string]bool{}},
		emailLimiter: &fakeLimiter{denied: map[string]bool{}},
		demoLimiter:  &fakeLimiter{denied: map[string]bool{}},
		spans:        tracetest.NewSpanRecorder(),
		metrics:      sdkmetric.NewManualReader(),
	}
	e.demo = &fakeDemo{tenants: e.tenants, users: users}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})

	permissions, err := app.NewPermissionMatrix(testRegistry)
	require.NoError(t, err)
	deps := app.Deps{
		Tenants:        e.tenants,
		Users:          users,
		Tokens:         e.tokens,
		Demo:           e.demo,
		Directory:      users,
		Hasher:         e.hasher,
		Issuer:         fakeIssuer{clock: clock},
		Setup:          e.setup,
		Permissions:    permissions,
		Modules:        e.modules,
		IPLimiter:      e.ipLimiter,
		EmailLimiter:   e.emailLimiter,
		DemoLimiter:    e.demoLimiter,
		DemoEnabled:    true,
		Now:            clock.Now,
		TracerProvider: tp,
		MeterProvider:  mp,
	}
	for _, opt := range opts {
		opt(&deps)
	}
	svc, err := app.NewService(deps)
	require.NoError(t, err)
	e.svc = svc
	return e
}

const (
	validNIT      = "900123456-8"
	validPassword = "clave-segura-123"
)

// seed stores a tenant and a user with the given email, password and role.
func (e *env) seed(t *testing.T, email, password string, role domain.Role) (*domain.Tenant, *domain.User) {
	t.Helper()
	nit, err := domain.ParseNIT(validNIT)
	require.NoError(t, err)
	tenant, err := domain.NewTenant("Ferretería El Tornillo", nit, "ferreteria", decimal.RequireFromString("0.19"), e.clock.Now())
	require.NoError(t, err)
	require.NoError(t, e.tenants.Create(t.Context(), tenant))
	return tenant, e.seedUser(t, tenant.ID(), email, password, role)
}

func (e *env) seedUser(t *testing.T, tenantID uuid.UUID, email, password string, role domain.Role) *domain.User {
	t.Helper()
	parsed, err := domain.ParseEmail(email)
	require.NoError(t, err)
	hash, err := e.hasher.Hash(password)
	require.NoError(t, err)
	user, err := domain.NewUser(tenantID, parsed, "Ana Gómez", hash, role, e.clock.Now())
	require.NoError(t, err)
	require.NoError(t, e.users.Create(t.Context(), user))
	return user
}

// loginCount returns the value of cuadra.identity.logins for result.
func (e *env) loginCount(t *testing.T, result string) int64 {
	t.Helper()
	return e.counter(t, "cuadra.identity.logins", "result", result)
}

// counter returns the sum of the data points of an int64 counter whose
// attribute key equals value (any data point when key is "").
func (e *env) counter(t *testing.T, name, key, value string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, e.metrics.Collect(t.Context(), &rm))
	var total int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "%s is not an int64 counter", name)
			for _, dp := range sum.DataPoints {
				if key == "" {
					total += dp.Value
					continue
				}
				if v, ok := dp.Attributes.Value(attribute.Key(key)); ok && v.AsString() == value {
					total += dp.Value
				}
			}
		}
	}
	return total
}

// span returns the only ended span named name.
func (e *env) span(t *testing.T, name string) sdktrace.ReadOnlySpan {
	t.Helper()
	var found []sdktrace.ReadOnlySpan
	for _, s := range e.spans.Ended() {
		if s.Name() == name {
			found = append(found, s)
		}
	}
	require.Len(t, found, 1, "spans named %s", name)
	return found[0]
}
