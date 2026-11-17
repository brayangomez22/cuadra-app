package httpadapter_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags"
	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/platform/flags/adapters/http"
)

// stubUseCases records the calls and answers what the test sets.
type stubUseCases struct {
	list []flags.Status
	set  flags.Status
	err  error

	userID, tenantID uuid.UUID
	key              string
	enabled          bool
	cleared          bool
}

func (s *stubUseCases) List(_ context.Context, userID uuid.UUID) ([]flags.Status, error) {
	s.userID = userID
	return s.list, s.err
}

func (s *stubUseCases) SetGlobal(_ context.Context, userID uuid.UUID, key string, enabled bool) (flags.Status, error) {
	s.userID, s.key, s.enabled = userID, key, enabled
	return s.set, s.err
}

func (s *stubUseCases) SetException(_ context.Context, userID, tenantID uuid.UUID, key string, enabled bool) error {
	s.userID, s.tenantID, s.key, s.enabled = userID, tenantID, key, enabled
	return s.err
}

func (s *stubUseCases) ClearException(_ context.Context, userID, tenantID uuid.UUID, key string) error {
	s.userID, s.tenantID, s.key, s.cleared = userID, tenantID, key, true
	return s.err
}

var operator = auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "owner"}

func serve(t *testing.T, uc *stubUseCases, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, slog.New(slog.DiscardHandler))
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req = req.WithContext(auth.WithPrincipal(req.Context(), operator))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestListFlags(t *testing.T) {
	t.Run("lista las flags con su valor global y sus excepciones", func(t *testing.T) {
		pilot := uuid.Must(uuid.NewV7())
		uc := &stubUseCases{list: []flags.Status{
			{Definition: flags.Definition{Key: "feedback", Description: "Botón"}, Global: false, Exceptions: []flags.Exception{
				{TenantID: pilot, TenantName: "Ferretería La Tuerca", Key: "feedback", Enabled: true},
			}},
			{Definition: flags.Definition{Key: "analytics", Description: "Analítica"}, Global: true},
		}}

		rec := serve(t, uc, http.MethodGet, "/api/v1/platform/flags", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"items": [
			{"key": "feedback", "description": "Botón", "enabled": false, "exceptions": [
				{"tenant_id": "`+pilot.String()+`", "tenant_name": "Ferretería La Tuerca", "enabled": true}
			]},
			{"key": "analytics", "description": "Analítica", "enabled": true, "exceptions": []}
		]}`, rec.Body.String())
		require.Equal(t, operator.UserID, uc.userID)
	})
}

func TestChangeFlags(t *testing.T) {
	tenant := uuid.Must(uuid.NewV7())

	t.Run("cambia el valor global con el usuario del token", func(t *testing.T) {
		uc := &stubUseCases{set: flags.Status{Definition: flags.Definition{Key: "feedback", Description: "Botón"}, Global: true}}

		rec := serve(t, uc, http.MethodPut, "/api/v1/platform/flags/feedback", `{"enabled": true}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"key": "feedback", "description": "Botón", "enabled": true}`, rec.Body.String())
		require.Equal(t, operator.UserID, uc.userID)
		require.True(t, uc.enabled)
	})

	t.Run("pone la excepción a la empresa de la ruta, no a la del token", func(t *testing.T) {
		uc := &stubUseCases{}

		rec := serve(t, uc, http.MethodPut, "/api/v1/platform/flags/feedback/tenants/"+tenant.String(), `{"enabled": false}`)

		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.Equal(t, tenant, uc.tenantID)
		require.Equal(t, "feedback", uc.key)
		require.False(t, uc.enabled)
	})

	t.Run("quita la excepción", func(t *testing.T) {
		uc := &stubUseCases{}

		rec := serve(t, uc, http.MethodDelete, "/api/v1/platform/flags/feedback/tenants/"+tenant.String(), "")

		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.True(t, uc.cleared)
		require.Equal(t, tenant, uc.tenantID)
	})

	errs := []struct {
		name string
		err  error
		code string
	}{
		{"quien no es operador recibe 404 como si la ruta no existiera", flags.ErrNotOperator, "not_found"},
		{"una flag desconocida responde 404", flags.ErrUnknownFlag, "unknown_flag"},
		{"una empresa desconocida responde 404", flags.ErrUnknownTenant, "unknown_tenant"},
	}
	for _, tc := range errs {
		t.Run(tc.name, func(t *testing.T) {
			rec := serve(t, &stubUseCases{err: tc.err}, http.MethodPut,
				"/api/v1/platform/flags/feedback/tenants/"+tenant.String(), `{"enabled": true}`)

			require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
			var body struct {
				Error struct{ Code, Message string } `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, tc.code, body.Error.Code)
			require.NotEmpty(t, body.Error.Message)
		})
	}
}
