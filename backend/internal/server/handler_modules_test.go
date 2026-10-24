package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
	"github.com/brayangomez22/cuadra-app/backend/internal/server/ports"
)

func TestHandlerModules(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())
	const variantsPath = "/api/v1/products/01920000-0000-7000-8000-000000000001/variants"
	const validVariant = `{"sku": "CAM-M", "cost": "10000.00", "price": "25000.00", "axis_values": {"talla": "M"}}`

	// setup mounts a stub of addProductVariant, which belongs to the
	// variants module.
	setup := func(t *testing.T, gate *fakeModules) (http.Handler, *bool) {
		t.Helper()
		reached := false
		cfg := testConfig(t)
		cfg.Modules = gate
		cfg.Routes = []func(*http.ServeMux){func(mux *http.ServeMux) {
			mux.HandleFunc("POST /api/v1/products/{productId}/variants", func(w http.ResponseWriter, _ *http.Request) {
				reached = true
				w.WriteHeader(http.StatusNoContent)
			})
		}}
		handler, err := newHandler(cfg)
		require.NoError(t, err)
		return handler, &reached
	}
	post := func(t *testing.T, handler http.Handler, role, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, variantsPath, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if role != "" {
			j, err := auth.NewJWT(testSecret, 15*time.Minute)
			require.NoError(t, err)
			tok, _, err := j.Issue(tenantID, uuid.Must(uuid.NewV7()), role)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	errorCode := func(t *testing.T, rec *httptest.ResponseRecorder) string {
		t.Helper()
		var body httpx.ErrorBody
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "body: %s", rec.Body.String())
		return body.Error.Code
	}

	t.Run("una ruta de un módulo inactivo responde 404", func(t *testing.T) {
		handler, reached := setup(t, newFakeModules(t))

		rec := post(t, handler, "admin", validVariant)

		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Equal(t, "not_found", errorCode(t, rec))
		require.False(t, *reached)
	})

	t.Run("una ruta de un módulo activo llega al handler", func(t *testing.T) {
		handler, reached := setup(t, newFakeModules(t, modules.Variants))

		rec := post(t, handler, "admin", validVariant)

		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.True(t, *reached)
	})

	t.Run("el 404 del módulo inactivo llega antes de validar el body", func(t *testing.T) {
		handler, _ := setup(t, newFakeModules(t))

		rec := post(t, handler, "admin", `{"sku": ""}`)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("sin token responde 401 antes de revisar el módulo", func(t *testing.T) {
		gate := newFakeModules(t)
		handler, _ := setup(t, gate)

		rec := post(t, handler, "", validVariant)

		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Zero(t, gate.checks)
	})

	t.Run("sin el permiso responde 403 antes de revisar el módulo", func(t *testing.T) {
		gate := newFakeModules(t)
		handler, _ := setup(t, gate)

		rec := post(t, handler, "cashier", validVariant)

		require.Equal(t, http.StatusForbidden, rec.Code)
		require.Zero(t, gate.checks)
	})

	t.Run("si no se puede consultar el módulo responde 500", func(t *testing.T) {
		gate := newFakeModules(t)
		gate.err = errors.New("db down")
		handler, reached := setup(t, gate)

		rec := post(t, handler, "admin", validVariant)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.Equal(t, "internal_error", errorCode(t, rec))
		require.False(t, *reached)
	})

	t.Run("las rutas del core no consultan los módulos", func(t *testing.T) {
		gate := newFakeModules(t)
		cfg := testConfig(t)
		cfg.Modules = gate
		handler, err := newHandler(cfg)
		require.NoError(t, err)

		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

		require.Zero(t, gate.checks)
	})
}

// unknownPermission is a permission matrix that does not register one
// permission.
type unknownPermission struct {
	*identity.Permissions
	permission string
}

func (u unknownPermission) Known(p string) bool { return p != u.permission && u.Permissions.Known(p) }

func TestIdentityRegistry(t *testing.T) {
	t.Run("pasa a identity los permisos de cada módulo con sus roles", func(t *testing.T) {
		registry, err := modules.NewRegistry(
			modules.Definition{ID: "variants", Name: "Variantes"},
			modules.Definition{
				ID: "lots_expiry", Name: "Lotes y vencimientos", DependsOn: []string{"variants"},
				Permissions: []modules.Permission{{ID: "lots:manage", Roles: []string{"admin", "warehouse"}}},
			},
		)
		require.NoError(t, err)

		permissions, err := identity.NewPermissions(ports.IdentityRegistry{Registry: registry})

		require.NoError(t, err)
		require.True(t, permissions.Known("lots:manage"))
		require.True(t, permissions.Authorize("warehouse", "lots:manage"))
		require.False(t, permissions.Authorize("cashier", "lots:manage"))
		require.True(t, permissions.Authorize("cashier", "sales:create"))
	})
}
