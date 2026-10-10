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
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// TestInventoryFlow runs the inventory through the whole stack: contract
// validation, permissions from x-permission, use cases and PostgreSQL with
// Row-Level Security.
func TestInventoryFlow(t *testing.T) {
	database := dbtest.New(t)
	tokens, err := auth.NewJWT(testSecret, 15*time.Minute)
	require.NoError(t, err)
	log := slog.New(slog.DiscardHandler)
	tp, mp := noop.NewTracerProvider(), metricnoop.NewMeterProvider()
	identityModule, err := identity.New(identity.Config{DB: database, Issuer: tokens, Logger: log, TracerProvider: tp, MeterProvider: mp})
	require.NoError(t, err)
	catalogModule, err := catalog.New(catalog.Config{DB: database, Authorize: identity.Authorize, Logger: log, TracerProvider: tp, MeterProvider: mp})
	require.NoError(t, err)
	inventoryModule, err := inventory.New(inventory.Config{DB: database, Authorize: identity.Authorize, Logger: log, TracerProvider: tp, MeterProvider: mp})
	require.NoError(t, err)
	handler, err := newHandler(handlerConfig{
		Log: log, TracerProvider: tp, DB: readyDB{},
		Verifier: tokens, Authorize: identity.Authorize,
		Routes: []func(*http.ServeMux){identityModule.RegisterRoutes, catalogModule.RegisterRoutes, inventoryModule.RegisterRoutes},
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
	id := func(t *testing.T, rec *httptest.ResponseRecorder) string {
		t.Helper()
		var body struct{ ID string }
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		return body.ID
	}

	signup := do(t, http.MethodPost, "/api/v1/auth/signup",
		`{"tenant_name":"Ferretería El Tornillo","nit":"890903938-8","name":"Ana Gómez","email":"ana@inventario.co","password":"clave-segura-123"}`, "")
	require.Equal(t, http.StatusCreated, signup.Code, signup.Body.String())
	var session struct {
		AccessToken string `json:"access_token"`
		User        struct {
			TenantID uuid.UUID `json:"tenant_id"`
		} `json:"user"`
	}
	require.NoError(t, json.Unmarshal(signup.Body.Bytes(), &session))
	ownerToken := session.AccessToken
	// Other roles of the same tenant: user management arrives in T41, so the
	// tokens are issued directly.
	cashierToken, _, err := tokens.Issue(session.User.TenantID, uuid.Must(uuid.NewV7()), "cashier")
	require.NoError(t, err)
	warehouseToken, _, err := tokens.Issue(session.User.TenantID, uuid.Must(uuid.NewV7()), "warehouse")
	require.NoError(t, err)

	product := do(t, http.MethodPost, "/api/v1/products",
		`{"sku":"cem-50","name":"Cemento gris 50kg","base_unit":"bulto","cost":"30000","price":"35000","tax_rate":"0.19"}`, ownerToken)
	require.Equal(t, http.StatusCreated, product.Code, product.Body.String())
	productID := id(t, product)

	var bodegaID, centroID string

	t.Run("el dueño crea sedes; el bodeguero no puede", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/locations", `{"name":"Bodega"}`, ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		bodegaID = id(t, rec)
		rec = do(t, http.MethodPost, "/api/v1/locations", `{"name":"Centro"}`, ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		centroID = id(t, rec)

		rec = do(t, http.MethodPost, "/api/v1/locations", `{"name":"Norte"}`, warehouseToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})

	receipt := func(qty, cost string) string {
		return `{"product_id":"` + productID + `","location_id":"` + bodegaID + `","quantity":"` + qty + `","unit_cost":"` + cost + `"}`
	}

	t.Run("registrar una entrada sin inventory:adjust responde 403", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/inventory/receipts", receipt("10", "30000"), cashierToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"forbidden"`)
	})

	t.Run("el bodeguero registra entradas sin ver el costo promedio", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/inventory/receipts", receipt("10", "30000"), warehouseToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"balance_after":"10"`)
		require.NotContains(t, rec.Body.String(), "cost")

		rec = do(t, http.MethodPost, "/api/v1/inventory/receipts", receipt("10", "32000"), warehouseToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	})

	t.Run("un traslado mayor al stock responde 422 insufficient_stock", func(t *testing.T) {
		body := `{"product_id":"` + productID + `","from_location_id":"` + bodegaID + `","to_location_id":"` + centroID + `","quantity":"21"}`
		rec := do(t, http.MethodPost, "/api/v1/inventory/transfers", body, warehouseToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "insufficient_stock")
	})

	t.Run("traslada y ajusta el stock", func(t *testing.T) {
		body := `{"product_id":"` + productID + `","from_location_id":"` + bodegaID + `","to_location_id":"` + centroID + `","quantity":"5"}`
		rec := do(t, http.MethodPost, "/api/v1/inventory/transfers", body, warehouseToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		body = `{"product_id":"` + productID + `","location_id":"` + bodegaID + `","quantity":"-0.5","reason":"Bulto roto"}`
		rec = do(t, http.MethodPost, "/api/v1/inventory/adjustments", body, warehouseToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	})

	t.Run("el dueño ve las existencias de la sede con su costo promedio", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/locations/"+bodegaID+"/stock", "", ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"quantity":"14.5"`)
		require.Contains(t, rec.Body.String(), `"average_cost":"31000.00"`)
	})

	t.Run("el cajero consulta el kardex, el más reciente primero, sin costos", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/products/"+productID+"/kardex?location_id="+bodegaID, "", cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var page struct {
			Items []struct{ Type, Quantity string }
			Total int
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		require.Equal(t, 4, page.Total)
		require.Equal(t, "adjustment", page.Items[0].Type)
		require.Equal(t, "-0.5", page.Items[0].Quantity)
		require.NotContains(t, rec.Body.String(), "cost")
	})

	t.Run("una cantidad con formato inválido responde 400 por el contrato", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/inventory/receipts", receipt("2,5", "30000"), ownerToken)
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})

	t.Run("un tenant_id en el body responde 400 por el contrato", func(t *testing.T) {
		body := strings.Replace(receipt("1", "1"), "{", `{"tenant_id":"`+uuid.Must(uuid.NewV7()).String()+`",`, 1)
		rec := do(t, http.MethodPost, "/api/v1/inventory/receipts", body, ownerToken)
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})
}
