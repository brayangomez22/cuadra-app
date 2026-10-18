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

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
)

// testModules wires the module service on database and the permission
// matrix of the core and Cuadra's modules, as main does.
func testModules(t *testing.T, database *db.DB) (*modules.Service, *identity.Permissions) {
	t.Helper()
	registry := modulesRegistry(t)
	svc, err := newModuleService(registry, database, slog.New(slog.DiscardHandler), noop.NewTracerProvider(), metricnoop.NewMeterProvider())
	require.NoError(t, err)
	permissions, err := identity.NewPermissions(identityRegistry{registry})
	require.NoError(t, err)
	return svc, permissions
}

// testSetup wires the sign-up's tenant setup with the embedded business
// templates, as main does.
func testSetup(t *testing.T, catalogModule *catalog.Module, moduleService *modules.Service) tenantSetup {
	t.Helper()
	setup, err := newTenantSetup(modulesRegistry(t), catalogModule, moduleService)
	require.NoError(t, err)
	return setup
}

// TestModulesFlow runs the modules through the whole stack: the owner turns
// variants on and off, the contract's x-module hides the operation, and the
// tenant's data stays.
func TestModulesFlow(t *testing.T) {
	database := dbtest.New(t)
	tokens, err := auth.NewJWT(testSecret, 15*time.Minute)
	require.NoError(t, err)
	log := slog.New(slog.DiscardHandler)
	moduleService, permissions := testModules(t, database)
	catalogModule, err := catalog.New(catalog.Config{
		DB: database, Authorize: permissions.Authorize, Logger: log,
		TracerProvider: noop.NewTracerProvider(), MeterProvider: metricnoop.NewMeterProvider(),
	})
	require.NoError(t, err)
	identityModule, err := identity.New(identity.Config{
		DB: database, Issuer: tokens, Setup: testSetup(t, catalogModule, moduleService),
		Permissions: permissions, Modules: moduleService, Logger: log,
		TracerProvider: noop.NewTracerProvider(), MeterProvider: metricnoop.NewMeterProvider(),
	})
	require.NoError(t, err)
	handler, err := newHandler(handlerConfig{
		Log: log, TracerProvider: noop.NewTracerProvider(), DB: readyDB{},
		Verifier: tokens, Permissions: permissions, Modules: moduleService,
		Routes: []func(*http.ServeMux){
			identityModule.RegisterRoutes, catalogModule.RegisterRoutes, moduleRoutes(moduleService, log),
		},
	})
	require.NoError(t, err)

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
	errorCode := func(t *testing.T, rec *httptest.ResponseRecorder) string {
		t.Helper()
		var body struct {
			Error struct{ Code string } `json:"error"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
		return body.Error.Code
	}
	type me struct {
		TenantID    uuid.UUID `json:"tenant_id"`
		Modules     []string  `json:"modules"`
		Permissions []string  `json:"permissions"`
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
		`{"tenant_name":"Moda Urbana","nit":"890903938-8","name":"Ana Gómez","email":"ana@modulos.co","password":"clave-segura-123","business_template":"generica"}`, "")
	require.Equal(t, http.StatusCreated, signup.Code, signup.Body.String())
	var session struct {
		AccessToken string `json:"access_token"`
		User        me     `json:"user"`
	}
	require.NoError(t, json.Unmarshal(signup.Body.Bytes(), &session))
	ownerToken := session.AccessToken
	// An admin of the same tenant: user management arrives in T41, so the
	// token is issued directly.
	adminToken, _, err := tokens.Issue(session.User.TenantID, uuid.Must(uuid.NewV7()), "admin")
	require.NoError(t, err)

	rec := do(t, http.MethodPost, "/api/v1/products",
		`{"sku":"cam-s","name":"Camiseta básica","base_unit_id":"`+unitID(t, do, ownerToken, "und")+
			`","cost":"12000","price":"25000","tax_rate":"0.19","attributes":{}}`, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var product struct {
		ID       string
		Variants []struct{ ID string }
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &product))
	variantsPath := "/api/v1/products/" + product.ID + "/variants"
	// Defining axes is the core's; adding variants that use them, the module's.
	rec = do(t, http.MethodPost, "/api/v1/variant-axes", `{"name":"talla","values":["S","M","L"]}`, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	const medium = `{"sku":"cam-m","cost":"12000","price":"26000","axis_values":{"talla":"M"}}`

	t.Run("un tenant nuevo no tiene módulos activos y no puede agregar variantes", func(t *testing.T) {
		require.Empty(t, session.User.Modules)
		require.Empty(t, getMe(t, ownerToken).Modules)

		rec := do(t, http.MethodPost, variantsPath, medium, ownerToken)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		require.Equal(t, "not_found", errorCode(t, rec))
	})

	t.Run("solo el owner ve y cambia los módulos", func(t *testing.T) {
		require.Equal(t, http.StatusForbidden, do(t, http.MethodGet, "/api/v1/modules", "", adminToken).Code)
		require.Equal(t, http.StatusForbidden, do(t, http.MethodPut, "/api/v1/modules/variants", `{"active":true}`, adminToken).Code)

		rec := do(t, http.MethodGet, "/api/v1/modules", "", ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"items":[{"id":"variants","name":"Variantes","depends_on":[],"active":false}]}`, rec.Body.String())
	})

	t.Run("con el módulo activo se agregan variantes y /me lo lista", func(t *testing.T) {
		rec := do(t, http.MethodPut, "/api/v1/modules/variants", `{"active":true}`, ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"id":"variants","name":"Variantes","depends_on":[],"active":true}`, rec.Body.String())
		require.Equal(t, []string{"variants"}, getMe(t, ownerToken).Modules)

		rec = do(t, http.MethodPut, variantsPath+"/"+product.Variants[0].ID,
			`{"sku":"cam-s","cost":"12000","price":"25000","axis_values":{"talla":"S"},"active":true}`, ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPost, variantsPath, medium, adminToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	})

	t.Run("desactivar el módulo no borra las variantes, pero no deja agregar más", func(t *testing.T) {
		rec := do(t, http.MethodPut, "/api/v1/modules/variants", `{"active":false}`, ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Empty(t, getMe(t, ownerToken).Modules)

		rec = do(t, http.MethodPost, variantsPath, `{"sku":"cam-l","cost":"12000","price":"27000","axis_values":{"talla":"L"}}`, ownerToken)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

		rec = do(t, http.MethodGet, "/api/v1/products/"+product.ID, "", ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var got struct {
			Variants []struct{ SKU string }
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Len(t, got.Variants, 2)

		// Editing a variant stays in the core: every product has its default one.
		rec = do(t, http.MethodPut, variantsPath+"/"+product.Variants[0].ID,
			`{"sku":"cam-s","cost":"12000","price":"25500","axis_values":{"talla":"S"},"active":true}`, ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	})

	t.Run("un módulo desconocido responde 404 unknown_module", func(t *testing.T) {
		rec := do(t, http.MethodPut, "/api/v1/modules/price_lists", `{"active":true}`, ownerToken)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		require.Equal(t, "unknown_module", errorCode(t, rec))
	})

	t.Run("los módulos de un tenant no se ven desde otro", func(t *testing.T) {
		rec := do(t, http.MethodPut, "/api/v1/modules/variants", `{"active":true}`, ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		other := do(t, http.MethodPost, "/api/v1/auth/signup",
			`{"tenant_name":"Ferretería El Tornillo","nit":"900123456-8","name":"Luis Pérez","email":"luis@modulos.co","password":"clave-segura-123","business_template":"ferreteria"}`, "")
		require.Equal(t, http.StatusCreated, other.Code, other.Body.String())
		var otherSession struct {
			AccessToken string `json:"access_token"`
		}
		require.NoError(t, json.Unmarshal(other.Body.Bytes(), &otherSession))

		require.Empty(t, getMe(t, otherSession.AccessToken).Modules)
		require.Equal(t, []string{"variants"}, getMe(t, ownerToken).Modules)
	})
}
