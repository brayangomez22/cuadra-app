package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

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
