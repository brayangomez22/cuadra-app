package httpadapter_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/platform/modules/adapters/http"
)

// stubUseCases records the calls and answers what the test sets.
type stubUseCases struct {
	list []modules.Status
	set  modules.Status
	err  error

	tenantID, userID uuid.UUID
	module           string
	active           bool
}

func (s *stubUseCases) List(_ context.Context, tenantID uuid.UUID) ([]modules.Status, error) {
	s.tenantID = tenantID
	return s.list, s.err
}

func (s *stubUseCases) SetActive(_ context.Context, tenantID, userID uuid.UUID, id string, active bool) (modules.Status, error) {
	s.tenantID, s.userID, s.module, s.active = tenantID, userID, id, active
	return s.set, s.err
}

var owner = auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "owner"}

func serve(t *testing.T, uc *stubUseCases, p *auth.Principal, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, slog.New(slog.DiscardHandler))
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if p != nil {
		req = req.WithContext(auth.WithPrincipal(req.Context(), *p))
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

type errorBody struct {
	Error struct {
		Code, Message string
		Details       map[string]string
	} `json:"error"`
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) errorBody {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	var body errorBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, code, body.Error.Code)
	require.NotEmpty(t, body.Error.Message)
	return body
}

func status(id, name string, active bool, dependsOn ...string) modules.Status {
	return modules.Status{Definition: modules.Definition{ID: id, Name: name, DependsOn: dependsOn}, Active: active}
}

func TestListModules(t *testing.T) {
	t.Run("lista los módulos con sus dependencias y su estado", func(t *testing.T) {
		uc := &stubUseCases{list: []modules.Status{
			status("variants", "Variantes", true),
			status("lots_expiry", "Lotes y vencimientos", false, "variants"),
		}}

		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/modules", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"items": [
			{"id": "variants", "name": "Variantes", "depends_on": [], "active": true},
			{"id": "lots_expiry", "name": "Lotes y vencimientos", "depends_on": ["variants"], "active": false}
		]}`, rec.Body.String())
		require.Equal(t, owner.TenantID, uc.tenantID)
	})

	t.Run("sin principal responde 401", func(t *testing.T) {
		rec := serve(t, &stubUseCases{}, nil, http.MethodGet, "/api/v1/modules", "")
		decodeError(t, rec, http.StatusUnauthorized, "unauthorized")
	})
}

func TestSetModuleActive(t *testing.T) {
	t.Run("activa un módulo con el tenant y el usuario del token", func(t *testing.T) {
		uc := &stubUseCases{set: status("work_orders", "Órdenes de trabajo", true)}

		rec := serve(t, uc, &owner, http.MethodPut, "/api/v1/modules/work_orders", `{"active": true}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"id": "work_orders", "name": "Órdenes de trabajo", "depends_on": [], "active": true}`, rec.Body.String())
		require.Equal(t, owner.TenantID, uc.tenantID)
		require.Equal(t, owner.UserID, uc.userID)
		require.Equal(t, "work_orders", uc.module)
		require.True(t, uc.active)
	})

	t.Run("desactiva un módulo", func(t *testing.T) {
		uc := &stubUseCases{set: status("variants", "Variantes", false)}

		rec := serve(t, uc, &owner, http.MethodPut, "/api/v1/modules/variants", `{"active": false}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.False(t, uc.active)
	})

	tests := []struct {
		name        string
		err         error
		status      int
		code        string
		wantModule  string
		wantMessage string
	}{
		{name: "un módulo desconocido responde 404", err: modules.ErrUnknownModule, status: http.StatusNotFound, code: "unknown_module"},
		{
			name:   "una dependencia inactiva responde 409 y nombra la dependencia",
			err:    fmt.Errorf("wrapped: %w", &modules.DependencyError{Err: modules.ErrDependencyInactive, Module: "variants", Name: "Variantes"}),
			status: http.StatusConflict, code: "module_dependency_inactive", wantModule: "variants", wantMessage: "Variantes",
		},
		{
			name:   "un módulo activo que depende de este responde 409 y lo nombra",
			err:    &modules.DependencyError{Err: modules.ErrRequiredBy, Module: "lots_expiry", Name: "Lotes y vencimientos"},
			status: http.StatusConflict, code: "module_required_by", wantModule: "lots_expiry", wantMessage: "Lotes y vencimientos",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(t, &stubUseCases{err: tt.err}, &owner, http.MethodPut, "/api/v1/modules/lots_expiry", `{"active": true}`)

			body := decodeError(t, rec, tt.status, tt.code)
			if tt.wantModule != "" {
				require.Equal(t, map[string]string{"module": tt.wantModule}, body.Error.Details)
				require.Contains(t, body.Error.Message, tt.wantMessage)
			}
		})
	}

	t.Run("sin principal responde 401", func(t *testing.T) {
		rec := serve(t, &stubUseCases{}, nil, http.MethodPut, "/api/v1/modules/variants", `{"active": true}`)
		decodeError(t, rec, http.StatusUnauthorized, "unauthorized")
	})
}
