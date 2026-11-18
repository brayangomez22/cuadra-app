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

// TestFeedbackAndAnalyticsFlow runs T40 through the whole stack: any user
// sends a comment, and /me brings the user's pseudonym only once the owner
// consented and the team turned the "analytics" flag on.
func TestFeedbackAndAnalyticsFlow(t *testing.T) {
	operatorID := uuid.Must(uuid.NewV7())
	var database *db.DB
	handler, tokens, _ := newAPIWith(t, func(cfg *server.Config) {
		cfg.PlatformOperators = []uuid.UUID{operatorID}
		database = cfg.DB
		cfg.AnalyticsSecret = []byte("analytics-key-only-for-tests-0123456789")
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
		ID          uuid.UUID `json:"id"`
		TenantID    uuid.UUID `json:"tenant_id"`
		Email       string    `json:"email"`
		Consent     bool      `json:"tenant_analytics_consent"`
		AnalyticsID *string   `json:"analytics_id"`
	}
	getMe := func(t *testing.T, bearer string) me {
		t.Helper()
		rec := do(t, http.MethodGet, "/api/v1/me", "", bearer)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var m me
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &m))
		return m
	}

	signup := do(t, http.MethodPost, "/api/v1/auth/signup",
		`{"tenant_name":"Calzado Andino","nit":"900123456-8","name":"Ana Gómez","email":"ana@andino.co","password":"clave-segura-123","business_template":"ropa_calzado"}`, "")
	require.Equal(t, http.StatusCreated, signup.Code, signup.Body.String())
	var session struct {
		AccessToken string `json:"access_token"`
		User        me     `json:"user"`
	}
	require.NoError(t, json.Unmarshal(signup.Body.Bytes(), &session))
	owner := session.AccessToken
	cashier := joinStaff(t, do, owner, "cashier").Token
	addOperator(t, database, session.User.TenantID, operatorID)
	operator, _, err := tokens.Issue(session.User.TenantID, operatorID, "owner")
	require.NoError(t, err)

	t.Run("cualquier usuario envía un comentario; de la pantalla queda la ruta", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/feedback",
			`{"message":"Me gustaría ver las tallas juntas","screen":"/productos?q=bota","app_version":"1.4.0"}`, cashier)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		rec = do(t, http.MethodPost, "/api/v1/feedback", `{"message":"   ","screen":"/","app_version":"1.4.0"}`, cashier)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	})

	t.Run("sin consentimiento ni flag, /me no trae seudónimo", func(t *testing.T) {
		m := getMe(t, owner)
		require.False(t, m.Consent)
		require.Nil(t, m.AnalyticsID)
	})

	t.Run("con el consentimiento del dueño y la flag prendida, /me trae un seudónimo que no es el usuario", func(t *testing.T) {
		rec := do(t, http.MethodPatch, "/api/v1/tenant", `{"analytics_consent": true}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Nil(t, getMe(t, owner).AnalyticsID, "falta la flag")

		// An exception, not the global value: the flags are global to the database the tests share.
		rec = do(t, http.MethodPut, "/api/v1/platform/flags/analytics/tenants/"+session.User.TenantID.String(), `{"enabled": true}`, operator)
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())

		m := getMe(t, owner)
		require.True(t, m.Consent)
		require.NotNil(t, m.AnalyticsID)
		require.Len(t, *m.AnalyticsID, 32)
		require.NotContains(t, *m.AnalyticsID, strings.ReplaceAll(m.ID.String(), "-", ""))
		require.NotContains(t, *m.AnalyticsID, "andino")
	})

	t.Run("el cajero no cambia el consentimiento", func(t *testing.T) {
		rec := do(t, http.MethodPatch, "/api/v1/tenant", `{"analytics_consent": false}`, cashier)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})
}
