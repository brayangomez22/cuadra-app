//go:build integration

package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// TestAuthFlow runs sign-up, /me, refresh, token reuse and logout through
// the whole stack: contract validation, auth middleware, use cases and
// PostgreSQL with Row-Level Security.
func TestAuthFlow(t *testing.T) {
	database := dbtest.New(t)
	tokens, err := auth.NewJWT(testSecret, 15*time.Minute)
	require.NoError(t, err)
	log := slog.New(slog.DiscardHandler)
	module, err := identity.New(identity.Config{
		DB: database, Issuer: tokens, Logger: log,
		TracerProvider: noop.NewTracerProvider(), MeterProvider: metricnoop.NewMeterProvider(),
	})
	require.NoError(t, err)
	handler, err := newHandler(handlerConfig{
		Log: log, TracerProvider: noop.NewTracerProvider(), DB: readyDB{},
		Verifier: tokens, Authorize: identity.Authorize,
		Routes: []func(*http.ServeMux){module.RegisterRoutes},
	})
	require.NoError(t, err)

	do := func(t *testing.T, method, path, body, bearer string, cookie *http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	type session struct {
		AccessToken string `json:"access_token"`
		User        struct {
			Role        string   `json:"role"`
			Email       string   `json:"email"`
			Permissions []string `json:"permissions"`
		} `json:"user"`
	}
	parse := func(t *testing.T, rec *httptest.ResponseRecorder) (session, *http.Cookie) {
		t.Helper()
		var s session
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &s), rec.Body.String())
		for _, c := range rec.Result().Cookies() {
			if c.Name == "cuadra_refresh" {
				return s, c
			}
		}
		t.Fatal("no refresh cookie")
		return s, nil
	}

	signup := do(t, http.MethodPost, "/api/v1/auth/signup",
		`{"tenant_name":"Ferretería El Tornillo","nit":"890903938-8","name":"Ana Gómez","email":"Ana@ElTornillo.co","password":"clave-segura-123"}`, "", nil)
	require.Equal(t, http.StatusCreated, signup.Code, signup.Body.String())
	first, firstCookie := parse(t, signup)
	require.Equal(t, "owner", first.User.Role)
	require.Equal(t, "ana@eltornillo.co", first.User.Email)

	t.Run("signup → me con el access token devuelve el owner con todos los permisos", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/me", "", first.AccessToken, nil)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var me struct {
			Role        string   `json:"role"`
			Permissions []string `json:"permissions"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &me))
		require.Equal(t, "owner", me.Role)
		require.Contains(t, me.Permissions, "users:manage")
	})

	t.Run("login con la contraseña errónea responde 401 invalid_credentials", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/auth/login", `{"email":"ana@eltornillo.co","password":"otra-clave"}`, "", nil)
		require.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "invalid_credentials")
	})

	var current *http.Cookie
	t.Run("login correcto y refresh rotan la cookie", func(t *testing.T) {
		login := do(t, http.MethodPost, "/api/v1/auth/login", `{"email":"ana@eltornillo.co","password":"clave-segura-123"}`, "", nil)
		require.Equal(t, http.StatusOK, login.Code, login.Body.String())
		_, loginCookie := parse(t, login)

		refresh := do(t, http.MethodPost, "/api/v1/auth/refresh", "", "", loginCookie)
		require.Equal(t, http.StatusOK, refresh.Code, refresh.Body.String())
		_, current = parse(t, refresh)
		require.NotEqual(t, loginCookie.Value, current.Value)

		// The login's token was rotated: presenting it again is a reuse and
		// kills the family, so the newest token stops working too.
		reuse := do(t, http.MethodPost, "/api/v1/auth/refresh", "", "", loginCookie)
		require.Equal(t, http.StatusUnauthorized, reuse.Code)
		require.Contains(t, reuse.Body.String(), "invalid_session")
		again := do(t, http.MethodPost, "/api/v1/auth/refresh", "", "", current)
		require.Equal(t, http.StatusUnauthorized, again.Code)
	})

	t.Run("logout revoca la sesión del signup", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/auth/logout", "", "", firstCookie)
		require.Equal(t, http.StatusNoContent, rec.Code)

		refresh := do(t, http.MethodPost, "/api/v1/auth/refresh", "", "", firstCookie)
		require.Equal(t, http.StatusUnauthorized, refresh.Code)
	})
}
