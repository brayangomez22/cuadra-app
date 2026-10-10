package httpadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

// stubUseCases answers with whatever each test sets.
type stubUseCases struct {
	signUp  func(app.SignUpInput) (app.Session, error)
	login   func(app.LoginInput) (app.Session, error)
	refresh func(string) (app.Session, error)
	logout  func(string) error
	me      func(tenantID, userID uuid.UUID) (app.Profile, error)
}

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

var now = time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)

func profile(t *testing.T, role domain.Role) app.Profile {
	t.Helper()
	nit, err := domain.ParseNIT("890903938-8")
	require.NoError(t, err)
	tenant, err := domain.NewTenant("Ferretería El Tornillo", nit, now)
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

func TestSignUp(t *testing.T) {
	body := `{"tenant_name":"Ferretería El Tornillo","nit":"890903938-8","name":"Ana Gómez","email":"ana@eltornillo.co","password":"clave-segura-123"}`

	t.Run("responde 201 con la sesión y la cookie de refresh segura", func(t *testing.T) {
		want := session(t)
		var got app.SignUpInput
		uc := &stubUseCases{signUp: func(in app.SignUpInput) (app.Session, error) { got = in; return want, nil }}

		rec := serve(t, uc, post("/api/v1/auth/signup", body))

		requireSession(t, rec, http.StatusCreated, want)
		require.Equal(t, app.SignUpInput{
			TenantName: "Ferretería El Tornillo", NIT: "890903938-8", OwnerName: "Ana Gómez",
			Email: "ana@eltornillo.co", Password: "clave-segura-123",
		}, got)
	})

	t.Run("traduce los errores de validación del dominio a 422 con su código", func(t *testing.T) {
		tests := []struct {
			err  error
			code string
		}{
			{err: domain.ErrInvalidEmail, code: "invalid_email"},
			{err: domain.ErrInvalidNIT, code: "invalid_nit"},
			{err: domain.ErrInvalidNITCheckDigit, code: "invalid_nit_check_digit"},
			{err: domain.ErrPasswordTooShort, code: "password_too_short"},
			{err: domain.ErrPasswordTooLong, code: "password_too_long"},
			{err: domain.ErrInvalidTenantName, code: "invalid_tenant_name"},
			{err: domain.ErrInvalidUserName, code: "invalid_user_name"},
		}
		for _, tt := range tests {
			t.Run(tt.code, func(t *testing.T) {
				uc := &stubUseCases{signUp: func(app.SignUpInput) (app.Session, error) { return app.Session{}, tt.err }}
				requireError(t, serve(t, uc, post("/api/v1/auth/signup", body)), http.StatusUnprocessableEntity, tt.code)
			})
		}
	})

	t.Run("un error inesperado responde 500 sin detalles", func(t *testing.T) {
		uc := &stubUseCases{signUp: func(app.SignUpInput) (app.Session, error) {
			return app.Session{}, errors.New("db: connection refused at 10.0.0.5")
		}}

		rec := serve(t, uc, post("/api/v1/auth/signup", body))

		requireError(t, rec, http.StatusInternalServerError, "internal_error")
		require.NotContains(t, rec.Body.String(), "10.0.0.5")
	})
}

func TestLogin(t *testing.T) {
	body := `{"email":"ana@eltornillo.co","password":"clave-segura-123"}`

	t.Run("responde 200 con la sesión y pasa la IP del cliente sin el puerto", func(t *testing.T) {
		want := session(t)
		var got app.LoginInput
		uc := &stubUseCases{login: func(in app.LoginInput) (app.Session, error) { got = in; return want, nil }}
		req := post("/api/v1/auth/login", body)
		req.RemoteAddr = "203.0.113.7:51234"

		rec := serve(t, uc, req)

		requireSession(t, rec, http.StatusOK, want)
		require.Equal(t, app.LoginInput{Email: "ana@eltornillo.co", Password: "clave-segura-123", ClientIP: "203.0.113.7"}, got)
	})

	t.Run("pasa el tenant elegido", func(t *testing.T) {
		tenantID := uuid.Must(uuid.NewV7())
		var got app.LoginInput
		uc := &stubUseCases{login: func(in app.LoginInput) (app.Session, error) { got = in; return session(t), nil }}

		rec := serve(t, uc, post("/api/v1/auth/login", `{"email":"ana@eltornillo.co","password":"x","tenant_id":"`+tenantID.String()+`"}`))

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, tenantID, got.TenantID)
	})

	t.Run("credenciales erróneas responden 401 invalid_credentials", func(t *testing.T) {
		uc := &stubUseCases{login: func(app.LoginInput) (app.Session, error) { return app.Session{}, domain.ErrInvalidCredentials }}
		requireError(t, serve(t, uc, post("/api/v1/auth/login", body)), http.StatusUnauthorized, "invalid_credentials")
	})

	t.Run("tenant suspendido responde 403 tenant_suspended", func(t *testing.T) {
		uc := &stubUseCases{login: func(app.LoginInput) (app.Session, error) { return app.Session{}, domain.ErrTenantSuspended }}
		requireError(t, serve(t, uc, post("/api/v1/auth/login", body)), http.StatusForbidden, "tenant_suspended")
	})

	t.Run("varias empresas responden 409 con la lista para elegir", func(t *testing.T) {
		a, b := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
		uc := &stubUseCases{login: func(app.LoginInput) (app.Session, error) {
			return app.Session{}, &app.TenantSelectionError{Tenants: []app.TenantOption{{ID: a, Name: "El Tornillo"}, {ID: b, Name: "La Tuerca"}}}
		}}

		rec := serve(t, uc, post("/api/v1/auth/login", body))

		requireError(t, rec, http.StatusConflict, "tenant_selection_required")
		var got struct {
			Tenants []struct{ ID, Name string } `json:"tenants"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Equal(t, []struct{ ID, Name string }{{ID: a.String(), Name: "El Tornillo"}, {ID: b.String(), Name: "La Tuerca"}}, got.Tenants)
	})

	t.Run("demasiados intentos responden 429 con Retry-After en segundos", func(t *testing.T) {
		uc := &stubUseCases{login: func(app.LoginInput) (app.Session, error) {
			return app.Session{}, &app.RateLimitError{RetryAfter: 90*time.Second + 300*time.Millisecond}
		}}

		rec := serve(t, uc, post("/api/v1/auth/login", body))

		requireError(t, rec, http.StatusTooManyRequests, "too_many_requests")
		require.Equal(t, "91", rec.Header().Get("Retry-After"))
	})
}

func TestRefreshSession(t *testing.T) {
	withCookie := func(value string) *http.Request {
		req := post("/api/v1/auth/refresh", "")
		req.AddCookie(&http.Cookie{Name: "cuadra_refresh", Value: value})
		return req
	}

	t.Run("rota la cookie y devuelve un access token nuevo", func(t *testing.T) {
		want := session(t)
		var got string
		uc := &stubUseCases{refresh: func(raw string) (app.Session, error) { got = raw; return want, nil }}

		rec := serve(t, uc, withCookie("old-refresh-token"))

		requireSession(t, rec, http.StatusOK, want)
		require.Equal(t, "old-refresh-token", got)
	})

	t.Run("sin cookie responde 401 invalid_session y borra la cookie", func(t *testing.T) {
		uc := &stubUseCases{refresh: func(string) (app.Session, error) {
			t.Fatal("must not be called without a cookie")
			return app.Session{}, nil
		}}

		rec := serve(t, uc, post("/api/v1/auth/refresh", ""))

		requireError(t, rec, http.StatusUnauthorized, "invalid_session")
		requireClearedCookie(t, rec)
	})

	t.Run("un token inválido, vencido o reutilizado responde 401 invalid_session y borra la cookie", func(t *testing.T) {
		for _, cause := range []error{domain.ErrInvalidRefreshToken, domain.ErrRefreshTokenExpired, domain.ErrRefreshTokenReused} {
			uc := &stubUseCases{refresh: func(string) (app.Session, error) { return app.Session{}, cause }}

			rec := serve(t, uc, withCookie("x"))

			requireError(t, rec, http.StatusUnauthorized, "invalid_session")
			requireClearedCookie(t, rec)
		}
	})

	t.Run("un error inesperado responde 500 y conserva la cookie", func(t *testing.T) {
		uc := &stubUseCases{refresh: func(string) (app.Session, error) { return app.Session{}, errors.New("db down") }}

		rec := serve(t, uc, withCookie("x"))

		requireError(t, rec, http.StatusInternalServerError, "internal_error")
		require.Empty(t, rec.Header().Values("Set-Cookie"))
	})
}

func TestLogout(t *testing.T) {
	t.Run("responde 204, revoca el token de la cookie y la borra", func(t *testing.T) {
		var got string
		uc := &stubUseCases{logout: func(raw string) error { got = raw; return nil }}
		req := post("/api/v1/auth/logout", "")
		req.AddCookie(&http.Cookie{Name: "cuadra_refresh", Value: "refresh-token"})

		rec := serve(t, uc, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
		require.Equal(t, "refresh-token", got)
		requireClearedCookie(t, rec)
	})

	t.Run("sin cookie también responde 204", func(t *testing.T) {
		uc := &stubUseCases{logout: func(raw string) error { require.Empty(t, raw); return nil }}

		rec := serve(t, uc, post("/api/v1/auth/logout", ""))

		require.Equal(t, http.StatusNoContent, rec.Code)
		requireClearedCookie(t, rec)
	})
}

func TestGetMe(t *testing.T) {
	t.Run("devuelve el usuario del token con su empresa y sus permisos", func(t *testing.T) {
		p := profile(t, domain.RoleCashier)
		var gotTenant, gotUser uuid.UUID
		uc := &stubUseCases{me: func(tenantID, userID uuid.UUID) (app.Profile, error) {
			gotTenant, gotUser = tenantID, userID
			return p, nil
		}}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		principal := auth.Principal{TenantID: p.Tenant.ID(), UserID: p.User.ID(), Role: "cashier"}
		req = req.WithContext(auth.WithPrincipal(req.Context(), principal))

		rec := serve(t, uc, req)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, p.Tenant.ID(), gotTenant)
		require.Equal(t, p.User.ID(), gotUser)
		require.JSONEq(t, `{
			"id": "`+p.User.ID().String()+`",
			"tenant_id": "`+p.Tenant.ID().String()+`",
			"tenant_name": "Ferretería El Tornillo",
			"name": "Ana Gómez",
			"email": "ana@eltornillo.co",
			"role": "cashier",
			"permissions": ["catalog:read", "inventory:read", "sales:create"]
		}`, rec.Body.String())
	})

	t.Run("sin usuario autenticado responde 401", func(t *testing.T) {
		uc := &stubUseCases{}
		requireError(t, serve(t, uc, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)), http.StatusUnauthorized, "unauthorized")
	})

	t.Run("un usuario que ya no existe o está inactivo responde 401", func(t *testing.T) {
		uc := &stubUseCases{me: func(uuid.UUID, uuid.UUID) (app.Profile, error) { return app.Profile{}, domain.ErrUserNotFound }}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "cashier"}))

		requireError(t, serve(t, uc, req), http.StatusUnauthorized, "unauthorized")
	})
}
