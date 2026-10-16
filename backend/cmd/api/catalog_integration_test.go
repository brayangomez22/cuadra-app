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
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// TestCatalogFlow runs the catalog through the whole stack: contract
// validation, permissions from x-permission, use cases and PostgreSQL with
// Row-Level Security.
func TestCatalogFlow(t *testing.T) {
	database := dbtest.New(t)
	tokens, err := auth.NewJWT(testSecret, 15*time.Minute)
	require.NoError(t, err)
	log := slog.New(slog.DiscardHandler)
	catalogModule, err := catalog.New(catalog.Config{
		DB: database, Authorize: identity.Authorize, Logger: log,
		TracerProvider: noop.NewTracerProvider(), MeterProvider: metricnoop.NewMeterProvider(),
	})
	require.NoError(t, err)
	identityModule, err := identity.New(identity.Config{
		DB: database, Issuer: tokens, Setup: tenantSetup{catalogModule}, Logger: log,
		TracerProvider: noop.NewTracerProvider(), MeterProvider: metricnoop.NewMeterProvider(),
	})
	require.NoError(t, err)
	handler, err := newHandler(handlerConfig{
		Log: log, TracerProvider: noop.NewTracerProvider(), DB: readyDB{},
		Verifier: tokens, Authorize: identity.Authorize,
		Routes: []func(*http.ServeMux){identityModule.RegisterRoutes, catalogModule.RegisterRoutes},
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

	signup := do(t, http.MethodPost, "/api/v1/auth/signup",
		`{"tenant_name":"Ferretería El Tornillo","nit":"890903938-8","name":"Ana Gómez","email":"ana@catalogo.co","password":"clave-segura-123"}`, "")
	require.Equal(t, http.StatusCreated, signup.Code, signup.Body.String())
	var session struct {
		AccessToken string `json:"access_token"`
		User        struct {
			TenantID uuid.UUID `json:"tenant_id"`
		} `json:"user"`
	}
	require.NoError(t, json.Unmarshal(signup.Body.Bytes(), &session))
	ownerToken := session.AccessToken
	// A cashier of the same tenant: user management arrives in T41, so the
	// token is issued directly.
	cashierToken, _, err := tokens.Issue(session.User.TenantID, uuid.Must(uuid.NewV7()), "cashier")
	require.NoError(t, err)

	const body = `{"sku":"tub-pvc-12","name":"Tubo PVC presión 1/2 pulgada","base_unit":"m","cost":"8333.33","price":"10000","tax_rate":"0.19"}`
	var productID string

	t.Run("el dueño crea un producto y recibe el dinero como string", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products", body, ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var p map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
		require.Equal(t, "TUB-PVC-12", p["sku"])
		require.Equal(t, "10000.00", p["price"])
		require.Equal(t, "8333.33", p["cost"])
		productID = p["id"].(string)
	})

	t.Run("crear producto sin permiso responde 403", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products", body, cashierToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"forbidden"`)
	})

	t.Run("el cajero busca el producto sin ver su costo", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/products?q=tubo+pvc", "", cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), productID)
		require.NotContains(t, rec.Body.String(), "cost")
	})

	t.Run("SKU duplicado responde 409 sku_taken", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products", body, ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "sku_taken")
	})

	t.Run("un precio con formato inválido responde 400 por el contrato", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products", strings.Replace(body, `"10000"`, `"10.000,50"`, 1), ownerToken)
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})

	var units struct {
		Items []struct {
			ID   string `json:"id"`
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"items"`
	}
	unitID := func(code string) string {
		for _, u := range units.Items {
			if u.Code == code {
				return u.ID
			}
		}
		t.Fatalf("no unit %q", code)
		return ""
	}

	t.Run("un tenant nuevo queda con las unidades por defecto", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/units", "", cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &units))
		got := map[string]string{}
		for _, u := range units.Items {
			got[u.Code] = u.Name
		}
		require.Equal(t, map[string]string{
			"und": "Unidad", "m": "Metro", "kg": "Kilogramo", "l": "Litro",
			"caja": "Caja", "rollo": "Rollo", "bulto": "Bulto", "galon": "Galón",
		}, got)
	})

	t.Run("el producto creado con el código deprecated lleva su unidad", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/products/"+productID, "", ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var p map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
		require.Equal(t, "m", p["base_unit"])
		require.Equal(t, unitID("m"), p["base_unit_id"])
		require.Equal(t, map[string]any{"id": unitID("m"), "code": "m", "name": "Metro"}, p["unit"])
	})

	t.Run("crea un producto con base_unit_id y lo devuelve con su unidad", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products",
			`{"sku":"cem-50","name":"Cemento gris 50 kg","base_unit_id":"`+unitID("bulto")+`","cost":"25000","price":"30000","tax_rate":"0.19"}`, ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var p map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
		require.Equal(t, unitID("bulto"), p["base_unit_id"])
		require.Equal(t, "bulto", p["base_unit"])
		require.Equal(t, map[string]any{"id": unitID("bulto"), "code": "bulto", "name": "Bulto"}, p["unit"])
	})

	t.Run("un producto sin unidad responde 422 unit_required", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products",
			`{"sku":"cam-s","name":"Camiseta básica","cost":"12000","price":"25000","tax_rate":"0.19"}`, ownerToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "unit_required")
	})

	t.Run("un producto con una unidad de otro tenant responde 422 unit_not_found", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products",
			`{"sku":"cam-s","name":"Camiseta básica","base_unit_id":"`+uuid.Must(uuid.NewV7()).String()+`","cost":"12000","price":"25000","tax_rate":"0.19"}`, ownerToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "unit_not_found")
	})

	t.Run("el dueño crea, edita y borra una unidad", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/units", `{"code":"Par","name":"Par"}`, ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var pair struct {
			ID   string `json:"id"`
			Code string `json:"code"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &pair))
		require.Equal(t, "par", pair.Code)

		rec = do(t, http.MethodPost, "/api/v1/units", `{"code":"par","name":"Pares"}`, ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "unit_code_taken")

		rec = do(t, http.MethodPut, "/api/v1/units/"+pair.ID, `{"code":"pares","name":"Pares"}`, ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"code":"pares"`)

		rec = do(t, http.MethodDelete, "/api/v1/units/"+pair.ID, "", ownerToken)
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	})

	t.Run("borrar una unidad en uso responde 409 unit_in_use", func(t *testing.T) {
		rec := do(t, http.MethodDelete, "/api/v1/units/"+unitID("m"), "", ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "unit_in_use")
	})

	t.Run("el cajero no crea unidades", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/units", `{"code":"docena","name":"Docena"}`, cashierToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})

	t.Run("limit mayor a 100 responde 400 por el contrato", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/products?limit=101", "", ownerToken)
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})
}
