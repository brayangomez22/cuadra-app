//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/server"
)

// TestFlagsFlow runs the feature flags through the whole stack: an operator
// releases a flag to one tenant (a pilot) and then to everyone, the tenants
// see it in /me, and the screen does not exist for anyone else.
func TestFlagsFlow(t *testing.T) {
	operatorID := uuid.Must(uuid.NewV7())
	var database *db.DB
	handler, tokens, _ := newAPIWith(t, func(cfg *server.Config) {
		cfg.PlatformOperators = []uuid.UUID{operatorID}
		database = cfg.DB
	})
	do := func(t *testing.T, method, path, body, bearer string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Authorization", "Bearer "+bearer)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	type me struct {
		TenantID uuid.UUID `json:"tenant_id"`
		Flags    []string  `json:"flags"`
		Operator bool      `json:"operator"`
	}
	signUp := func(t *testing.T, tenant, email, template string) (string, me) {
		t.Helper()
		rec := do(t, http.MethodPost, "/api/v1/auth/signup", `{"tenant_name":"`+tenant+`","nit":"900123456-8","name":"Ana Gómez","email":"`+
			email+`","password":"clave-segura-123","business_template":"`+template+`"}`, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var s struct {
			AccessToken string `json:"access_token"`
			User        me     `json:"user"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &s))
		return s.AccessToken, s.User
	}
	getMe := func(t *testing.T, bearer string) me {
		t.Helper()
		rec := do(t, http.MethodGet, "/api/v1/me", "", bearer)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var m me
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &m))
		return m
	}

	pilotToken, pilot := signUp(t, "Ferretería La Tuerca", "ana@latuerca.co", "ferreteria")
	otherToken, _ := signUp(t, "Calzado Andino", "ana@andino.co", "ropa_calzado")
	addOperator(t, database, pilot.TenantID, operatorID)
	operator, _, err := tokens.Issue(pilot.TenantID, operatorID, "owner")
	require.NoError(t, err)
	// The global values live in the database the API tests share: leave it as found.
	t.Cleanup(func() {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/platform/flags/feedback", strings.NewReader(`{"enabled": false}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+operator)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	})

	t.Run("una empresa nueva no tiene flags y su dueño no es operador", func(t *testing.T) {
		require.Equal(t, me{TenantID: pilot.TenantID, Flags: []string{}}, pilot)
	})

	t.Run("quien no es operador no encuentra las flags", func(t *testing.T) {
		require.Equal(t, http.StatusNotFound, do(t, http.MethodGet, "/api/v1/platform/flags", "", pilotToken).Code)
		rec := do(t, http.MethodPut, "/api/v1/platform/flags/feedback", `{"enabled": true}`, pilotToken)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("el operador libera una flag solo al piloto y después a todos", func(t *testing.T) {
		rec := do(t, http.MethodPut, "/api/v1/platform/flags/feedback/tenants/"+pilot.TenantID.String(), `{"enabled": true}`, operator)
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.Equal(t, []string{"feedback"}, getMe(t, pilotToken).Flags)
		require.Empty(t, getMe(t, otherToken).Flags)

		rec = do(t, http.MethodPut, "/api/v1/platform/flags/feedback", `{"enabled": true}`, operator)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, []string{"feedback"}, getMe(t, otherToken).Flags)

		rec = do(t, http.MethodGet, "/api/v1/platform/flags", "", operator)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"tenant_name":"Ferretería La Tuerca"`)
	})

	t.Run("una excepción apaga la flag para una empresa y quitarla la devuelve al valor global", func(t *testing.T) {
		path := "/api/v1/platform/flags/feedback/tenants/" + pilot.TenantID.String()
		require.Equal(t, http.StatusNoContent, do(t, http.MethodPut, path, `{"enabled": false}`, operator).Code)
		require.Empty(t, getMe(t, pilotToken).Flags)

		require.Equal(t, http.StatusNoContent, do(t, http.MethodDelete, path, "", operator).Code)
		require.Equal(t, []string{"feedback"}, getMe(t, pilotToken).Flags)
	})

	t.Run("una flag o una empresa desconocidas responden 404", func(t *testing.T) {
		rec := do(t, http.MethodPut, "/api/v1/platform/flags/nope", `{"enabled": true}`, operator)
		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Contains(t, rec.Body.String(), "unknown_flag")
		rec = do(t, http.MethodPut, "/api/v1/platform/flags/feedback/tenants/"+uuid.Must(uuid.NewV7()).String(), `{"enabled": true}`, operator)
		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Contains(t, rec.Body.String(), "unknown_tenant")
	})
}
