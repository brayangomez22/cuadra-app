package httpadapter_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

// stubUseCases answers with whatever each test sets.
type stubUseCases struct {
	signUp  func(app.SignUpInput) (app.Session, error)
	login   func(app.LoginInput) (app.Session, error)
	refresh func(string) (app.Session, error)
	logout  func(string) error
	me      func(tenantID, userID uuid.UUID) (app.Profile, error)
	update  func(tenantID, userID uuid.UUID, in app.TenantUpdate) (*domain.Tenant, error)
	// templates is what BusinessTemplates answers.
	templates []app.BusinessTemplate
}

func (s *stubUseCases) BusinessTemplates() []app.BusinessTemplate { return s.templates }

func (s *stubUseCases) SignUp(_ context.Context, in app.SignUpInput) (app.Session, error) {
	return s.signUp(in)
}

func (s *stubUseCases) Login(_ context.Context, in app.LoginInput) (app.Session, error) {
	return s.login(in)
}

func (s *stubUseCases) Refresh(_ context.Context, raw string) (app.Session, error) {
	return s.refresh(raw)
}

func (s *stubUseCases) Logout(_ context.Context, raw string) error { return s.logout(raw) }

func (s *stubUseCases) Me(_ context.Context, tenantID, userID uuid.UUID) (app.Profile, error) {
	return s.me(tenantID, userID)
}

func (s *stubUseCases) UpdateTenant(_ context.Context, tenantID, userID uuid.UUID, in app.TenantUpdate) (*domain.Tenant, error) {
	return s.update(tenantID, userID, in)
}

var now = time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)

func profile(t *testing.T, role domain.Role) app.Profile {
	t.Helper()
	nit, err := domain.ParseNIT("900123456-8")
	require.NoError(t, err)
	tenant, err := domain.NewTenant("Ferretería El Tornillo", nit, "ferreteria", decimal.RequireFromString("0.19"), now)
	require.NoError(t, err)
	email, err := domain.ParseEmail("ana@eltornillo.co")
	require.NoError(t, err)
	user, err := domain.NewUser(tenant.ID(), email, "Ana Gómez", "hash", role, now)
	require.NoError(t, err)
	return app.Profile{User: user, Tenant: tenant}
}

func session(t *testing.T) app.Session {
	t.Helper()
	return app.Session{
		AccessToken:      "access-token",
		AccessExpiresAt:  now.Add(15 * time.Minute),
		RefreshToken:     "refresh-token",
		RefreshExpiresAt: now.Add(domain.RefreshTokenTTL),
		Profile:          profile(t, domain.RoleOwner),
	}
}

func serve(t *testing.T, uc httpadapter.UseCases, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, slog.New(slog.DiscardHandler), httpadapter.WithClock(func() time.Time { return now }))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func post(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func requireError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, code, body.Error.Code)
	require.NotEmpty(t, body.Error.Message)
}

// refreshCookie returns the cuadra_refresh cookie set by the response.
func refreshCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == "cuadra_refresh" {
			return c
		}
	}
	t.Fatalf("no cuadra_refresh cookie in %v", rec.Header().Values("Set-Cookie"))
	return nil
}

func requireSecureCookie(t *testing.T, c *http.Cookie) {
	t.Helper()
	require.True(t, c.HttpOnly)
	require.True(t, c.Secure)
	require.Equal(t, http.SameSiteStrictMode, c.SameSite)
	require.Equal(t, "/api/v1/auth", c.Path)
}

func requireClearedCookie(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	c := refreshCookie(t, rec)
	requireSecureCookie(t, c)
	require.Empty(t, c.Value)
	require.Negative(t, c.MaxAge)
}

func requireSession(t *testing.T, rec *httptest.ResponseRecorder, status int, want app.Session) {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	var body struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		User        struct {
			ID, TenantID, TenantName, Name, Email, Role string
			Permissions                                 []string
		} `json:"user"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, want.AccessToken, body.AccessToken)
	require.Equal(t, "Bearer", body.TokenType)
	require.Equal(t, 900, body.ExpiresIn)
	require.NotContains(t, rec.Body.String(), want.RefreshToken, "the refresh token travels only in the cookie")

	c := refreshCookie(t, rec)
	requireSecureCookie(t, c)
	require.Equal(t, want.RefreshToken, c.Value)
	require.Equal(t, int(domain.RefreshTokenTTL.Seconds()), c.MaxAge)
}
