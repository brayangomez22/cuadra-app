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

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// TestSignUpTemplates signs a business up with each business template
// through the whole stack and checks that the tenant starts with exactly
// what its template declares.
func TestSignUpTemplates(t *testing.T) {
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
	setup := testSetup(t, catalogModule, moduleService)
	identityModule, err := identity.New(identity.Config{
		DB: database, Issuer: tokens, Setup: setup,
		Permissions: permissions, Modules: moduleService, Logger: log,
		TracerProvider: noop.NewTracerProvider(), MeterProvider: metricnoop.NewMeterProvider(),
	})
	require.NoError(t, err)
	handler, err := newHandler(handlerConfig{
		Log: log, TracerProvider: noop.NewTracerProvider(), DB: readyDB{},
		Verifier: tokens, Permissions: permissions, Modules: moduleService,
		Routes: []func(*http.ServeMux){identityModule.RegisterRoutes, catalogModule.RegisterRoutes},
	})
	require.NoError(t, err)

	do := func(t *testing.T, method, path, body, bearer string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	get := func(t *testing.T, path, bearer string, out any) {
		t.Helper()
		rec := do(t, http.MethodGet, path, "", bearer)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), out))
	}
	signUp := func(t *testing.T, email, template string) *httptest.ResponseRecorder {
		t.Helper()
		body := `{"tenant_name":"Mi negocio","nit":"890903938-8","name":"Ana Gómez","email":"` + email + `","password":"clave-segura-123"`
		if template != "" {
			body += `,"business_template":"` + template + `"`
		}
		return do(t, http.MethodPost, "/api/v1/auth/signup", body+"}", "")
	}

	t.Run("lista las plantillas sin autenticación", func(t *testing.T) {
		var list struct {
			Items []struct{ ID, Name, Description string } `json:"items"`
		}
		get(t, "/api/v1/business-templates", "", &list)
		ids := []string{}
		for _, item := range list.Items {
			ids = append(ids, item.ID)
			require.NotEmpty(t, item.Name)
			require.NotEmpty(t, item.Description)
		}
		require.ElementsMatch(t, []string{"generica", "ferreteria", "tienda", "drogueria", "papeleria", "ropa_calzado"}, ids)
	})

	for _, tpl := range setup.templates.List() {
		t.Run("aplicar la plantilla siembra exactamente sus datos: "+tpl.ID, func(t *testing.T) {
			rec := signUp(t, "ana@"+strings.ReplaceAll(tpl.ID, "_", "")+".co", tpl.ID)
			require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
			var session struct {
				AccessToken string `json:"access_token"`
				User        struct {
					BusinessTemplate string   `json:"business_template"`
					Modules          []string `json:"modules"`
				} `json:"user"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
			require.Equal(t, tpl.ID, session.User.BusinessTemplate)
			require.ElementsMatch(t, tpl.Modules, session.User.Modules)
			token := session.AccessToken

			var me struct {
				BusinessTemplate string   `json:"business_template"`
				DefaultTaxRate   string   `json:"default_tax_rate"`
				Modules          []string `json:"modules"`
			}
			get(t, "/api/v1/me", token, &me)
			require.Equal(t, tpl.ID, me.BusinessTemplate)
			require.Equal(t, tpl.DefaultTaxRate.String(), me.DefaultTaxRate)
			require.ElementsMatch(t, tpl.Modules, me.Modules)

			var units struct {
				Items []struct{ Code, Name string } `json:"items"`
			}
			get(t, "/api/v1/units", token, &units)
			wantUnits, gotUnits := map[string]string{}, map[string]string{}
			for _, u := range tpl.Units {
				wantUnits[u.Code] = u.Name
			}
			for _, u := range units.Items {
				gotUnits[u.Code] = u.Name
			}
			require.Equal(t, wantUnits, gotUnits)

			var categories struct {
				Items []struct {
					Name     string  `json:"name"`
					ParentID *string `json:"parent_id"`
				} `json:"items"`
			}
			get(t, "/api/v1/categories", token, &categories)
			gotCategories := []string{}
			for _, c := range categories.Items {
				require.Nil(t, c.ParentID)
				gotCategories = append(gotCategories, c.Name)
			}
			require.ElementsMatch(t, tpl.Categories, gotCategories)

			var attributes struct {
				Items []struct {
					Key, Label, Type string
					Options          []string
					Required         bool
				} `json:"items"`
			}
			get(t, "/api/v1/product-attributes", token, &attributes)
			require.Len(t, attributes.Items, len(tpl.Attributes))
			for _, want := range tpl.Attributes {
				found := false
				for _, got := range attributes.Items {
					if got.Key == want.Key {
						found = true
						require.Equal(t, want.Label, got.Label)
						require.Equal(t, want.Type, got.Type)
						require.Equal(t, len(want.Options), len(got.Options))
						require.Equal(t, want.Required, got.Required)
					}
				}
				require.True(t, found, "attribute %s", want.Key)
			}

			var axes struct {
				Items []struct {
					Name   string   `json:"name"`
					Values []string `json:"values"`
				} `json:"items"`
			}
			get(t, "/api/v1/variant-axes", token, &axes)
			wantAxes, gotAxes := map[string][]string{}, map[string][]string{}
			for _, a := range tpl.VariantAxes {
				wantAxes[strings.ToLower(a.Name)] = a.Values
			}
			for _, a := range axes.Items {
				gotAxes[a.Name] = a.Values
			}
			require.Equal(t, wantAxes, gotAxes)
		})
	}

	t.Run("sin plantilla, la empresa empieza con la genérica", func(t *testing.T) {
		rec := signUp(t, "ana@sinplantilla.co", "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"business_template":"generica"`)
	})

	t.Run("un registro con una plantilla inexistente responde 422 y no crea el tenant", func(t *testing.T) {
		rec := signUp(t, "ana@inexistente.co", "panaderia")
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "unknown_business_template")

		login := do(t, http.MethodPost, "/api/v1/auth/login", `{"email":"ana@inexistente.co","password":"clave-segura-123"}`, "")
		require.Equal(t, http.StatusUnauthorized, login.Code, "no tenant or user was created")
	})

}
