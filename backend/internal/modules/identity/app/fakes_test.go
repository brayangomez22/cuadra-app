package app_test

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// In-memory fakes of the identity ports. They copy entities in and out, like
// a database would, so a use case cannot change stored state by mutating an
// entity it did not save.

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time          { return c.now }
func (c *fakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

type fakeTenants struct {
	mu   sync.Mutex
	byID map[uuid.UUID]*domain.Tenant
	// users receives the owner on CreateWithOwner.
	users *fakeUsers
}

func cloneTenant(t *domain.Tenant) *domain.Tenant {
	c, err := domain.RehydrateTenant(domain.TenantSnapshot{ID: t.ID(), Name: t.Name(), NIT: t.NIT(), Status: t.Status(), CreatedAt: t.CreatedAt()})
	if err != nil {
		panic(err)
	}
	return c
}

func (f *fakeTenants) Create(_ context.Context, t *domain.Tenant) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[t.ID()] = cloneTenant(t)
	return nil
}

func (f *fakeTenants) GetByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrTenantNotFound
	}
	return cloneTenant(t), nil
}

func (f *fakeTenants) Update(_ context.Context, t *domain.Tenant) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.byID[t.ID()]; !ok {
		return domain.ErrTenantNotFound
	}
	f.byID[t.ID()] = cloneTenant(t)
	return nil
}

func (f *fakeTenants) CreateWithOwner(ctx context.Context, t *domain.Tenant, owner *domain.User) error {
	if err := f.Create(ctx, t); err != nil {
		return err
	}
	return f.users.Create(ctx, owner)
}

type fakeUsers struct {
	mu   sync.Mutex
	byID map[uuid.UUID]*domain.User
}

func cloneUser(u *domain.User) *domain.User {
	c, err := domain.RehydrateUser(domain.UserSnapshot{
		ID: u.ID(), TenantID: u.TenantID(), Email: u.Email(), Name: u.Name(),
		PasswordHash: u.PasswordHash(), Role: u.Role(), Active: u.IsActive(), CreatedAt: u.CreatedAt(),
	})
	if err != nil {
		panic(err)
	}
	return c
}

func (f *fakeUsers) Create(_ context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, other := range f.byID {
		if other.TenantID() == u.TenantID() && other.Email() == u.Email() {
			return domain.ErrEmailTaken
		}
	}
	f.byID[u.ID()] = cloneUser(u)
	return nil
}

func (f *fakeUsers) GetByID(_ context.Context, tenantID, id uuid.UUID) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok || u.TenantID() != tenantID {
		return nil, domain.ErrUserNotFound
	}
	return cloneUser(u), nil
}

func (f *fakeUsers) GetByEmail(_ context.Context, tenantID uuid.UUID, email domain.Email) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.byID {
		if u.TenantID() == tenantID && u.Email() == email {
			return cloneUser(u), nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (f *fakeUsers) Update(_ context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if old, ok := f.byID[u.ID()]; !ok || old.TenantID() != u.TenantID() {
		return domain.ErrUserNotFound
	}
	f.byID[u.ID()] = cloneUser(u)
	return nil
}

// FindByEmail implements domain.LoginDirectory over the same users.
func (f *fakeUsers) FindByEmail(_ context.Context, email domain.Email) ([]domain.LoginCandidate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.LoginCandidate
	for _, u := range f.byID {
		if u.Email() == email {
			out = append(out, domain.LoginCandidate{TenantID: u.TenantID(), UserID: u.ID()})
		}
	}
	slices.SortFunc(out, func(a, b domain.LoginCandidate) int { return bytes.Compare(a.TenantID[:], b.TenantID[:]) })
	if len(out) > domain.MaxLoginCandidates {
		out = out[:domain.MaxLoginCandidates]
	}
	return out, nil
}

type fakeTokens struct {
	mu   sync.Mutex
	byID map[uuid.UUID]*domain.RefreshToken
	// beforeRotate, when set, runs inside Rotate before the check: it
	// simulates a concurrent rotation of the same token.
	beforeRotate func()
}

func cloneToken(t *domain.RefreshToken) *domain.RefreshToken {
	c, err := domain.RehydrateRefreshToken(domain.RefreshTokenSnapshot{
		ID: t.ID(), TenantID: t.TenantID(), UserID: t.UserID(), FamilyID: t.FamilyID(),
		Hash: t.Hash(), ExpiresAt: t.ExpiresAt(), CreatedAt: t.CreatedAt(), RevokedAt: t.RevokedAt(),
	})
	if err != nil {
		panic(err)
	}
	return c
}

func (f *fakeTokens) Create(_ context.Context, t *domain.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[t.ID()] = cloneToken(t)
	return nil
}

func (f *fakeTokens) GetByHash(_ context.Context, tenantID uuid.UUID, hash []byte) (*domain.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, t := range f.byID {
		if t.TenantID() == tenantID && bytes.Equal(t.Hash(), hash) {
			return cloneToken(t), nil
		}
	}
	return nil, domain.ErrRefreshTokenNotFound
}

func (f *fakeTokens) Rotate(_ context.Context, old, next *domain.RefreshToken) error {
	if f.beforeRotate != nil {
		f.beforeRotate()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	stored, ok := f.byID[old.ID()]
	if !ok || stored.IsRevoked() {
		return domain.ErrRefreshTokenReused
	}
	f.byID[old.ID()] = cloneToken(old)
	f.byID[next.ID()] = cloneToken(next)
	return nil
}

func (f *fakeTokens) RevokeFamily(_ context.Context, tenantID, familyID uuid.UUID, now time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var revoked int64
	for id, t := range f.byID {
		if t.TenantID() != tenantID || t.FamilyID() != familyID || t.IsRevoked() {
			continue
		}
		revokedAt := now
		f.byID[id] = mustRehydrateToken(t, &revokedAt)
		revoked++
	}
	return revoked, nil
}

func mustRehydrateToken(t *domain.RefreshToken, revokedAt *time.Time) *domain.RefreshToken {
	c, err := domain.RehydrateRefreshToken(domain.RefreshTokenSnapshot{
		ID: t.ID(), TenantID: t.TenantID(), UserID: t.UserID(), FamilyID: t.FamilyID(),
		Hash: t.Hash(), ExpiresAt: t.ExpiresAt(), CreatedAt: t.CreatedAt(), RevokedAt: revokedAt,
	})
	if err != nil {
		panic(err)
	}
	return c
}

// family returns the stored tokens of a family.
func (f *fakeTokens) family(familyID uuid.UUID) []*domain.RefreshToken {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*domain.RefreshToken
	for _, t := range f.byID {
		if t.FamilyID() == familyID {
			out = append(out, cloneToken(t))
		}
	}
	return out
}

// fakeHasher "hashes" by prefixing, and counts verifications to check that
// a login with an unknown email still pays for one.
type fakeHasher struct {
	mu       sync.Mutex
	verifies int
}

func (h *fakeHasher) Hash(plain string) (string, error) { return "hash:" + plain, nil }

func (h *fakeHasher) Verify(plain, hash string) (bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.verifies++
	if !strings.HasPrefix(hash, "hash:") {
		return false, fmt.Errorf("malformed hash")
	}
	return hash == "hash:"+plain, nil
}

func (h *fakeHasher) Verifies() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.verifies
}

// fakeIssuer issues readable access tokens: "access:<tenant>:<user>:<role>".
type fakeIssuer struct{ clock *fakeClock }

func (i fakeIssuer) Issue(tenantID, userID uuid.UUID, role string) (string, time.Time, error) {
	return fmt.Sprintf("access:%s:%s:%s", tenantID, userID, role), i.clock.Now().Add(15 * time.Minute), nil
}

// fakeLimiter allows every key except the denied ones, and records the keys
// it was asked about.
type fakeLimiter struct {
	mu     sync.Mutex
	denied map[string]bool
	keys   []string
}

func (l *fakeLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.keys = append(l.keys, key)
	if l.denied[key] {
		return false, time.Minute
	}
	return true, 0
}

// env is a Service wired to fakes, with recorders for spans and metrics.
type env struct {
	svc          *app.Service
	clock        *fakeClock
	tenants      *fakeTenants
	users        *fakeUsers
	tokens       *fakeTokens
	hasher       *fakeHasher
	ipLimiter    *fakeLimiter
	emailLimiter *fakeLimiter
	spans        *tracetest.SpanRecorder
	metrics      *sdkmetric.ManualReader
}

func newEnv(t *testing.T) *env {
	t.Helper()
	clock := &fakeClock{now: time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)}
	users := &fakeUsers{byID: map[uuid.UUID]*domain.User{}}
	e := &env{
		clock:        clock,
		users:        users,
		tenants:      &fakeTenants{byID: map[uuid.UUID]*domain.Tenant{}, users: users},
		tokens:       &fakeTokens{byID: map[uuid.UUID]*domain.RefreshToken{}},
		hasher:       &fakeHasher{},
		ipLimiter:    &fakeLimiter{denied: map[string]bool{}},
		emailLimiter: &fakeLimiter{denied: map[string]bool{}},
		spans:        tracetest.NewSpanRecorder(),
		metrics:      sdkmetric.NewManualReader(),
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(e.spans))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(e.metrics))
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
	})

	svc, err := app.NewService(app.Deps{
		Tenants:        e.tenants,
		Users:          users,
		Tokens:         e.tokens,
		Directory:      users,
		Hasher:         e.hasher,
		Issuer:         fakeIssuer{clock: clock},
		IPLimiter:      e.ipLimiter,
		EmailLimiter:   e.emailLimiter,
		Now:            clock.Now,
		TracerProvider: tp,
		MeterProvider:  mp,
	})
	require.NoError(t, err)
	e.svc = svc
	return e
}

const (
	validNIT      = "890903938-8"
	validPassword = "clave-segura-123"
)

// seed stores a tenant and a user with the given email, password and role.
func (e *env) seed(t *testing.T, email, password string, role domain.Role) (*domain.Tenant, *domain.User) {
	t.Helper()
	nit, err := domain.ParseNIT(validNIT)
	require.NoError(t, err)
	tenant, err := domain.NewTenant("Ferretería El Tornillo", nit, e.clock.Now())
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
