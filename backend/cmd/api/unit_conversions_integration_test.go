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
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// TestUnitConversionsFlow sells and receives in alternate units through the
// whole stack: the catalog keeps the units of each variant, the inventory
// receives and the sales sell in them, and the stock stays in the base unit.
func TestUnitConversionsFlow(t *testing.T) {
	database := dbtest.New(t)
	moduleService, permissions := testModules(t, database)
	tokens, err := auth.NewJWT(testSecret, 15*time.Minute)
	require.NoError(t, err)
	log := slog.New(slog.DiscardHandler)
	tp, mp := noop.NewTracerProvider(), metricnoop.NewMeterProvider()
	catalogModule, err := catalog.New(catalog.Config{DB: database, Authorize: permissions.Authorize, Logger: log, TracerProvider: tp, MeterProvider: mp})
	require.NoError(t, err)
	identityModule, err := identity.New(identity.Config{
		DB: database, Issuer: tokens, Setup: testSetup(t, catalogModule, moduleService),
		Permissions: permissions, Modules: moduleService, Logger: log, TracerProvider: tp, MeterProvider: mp,
	})
	require.NoError(t, err)
	inventoryModule, err := inventory.New(inventory.Config{
		DB: database, Authorize: permissions.Authorize, Variants: inventoryCatalog{catalogModule},
		Logger: log, TracerProvider: tp, MeterProvider: mp,
	})
	require.NoError(t, err)
	salesModule, err := sales.New(sales.Config{
		DB: database, Authorize: permissions.Authorize,
		Catalog: salesCatalog{catalogModule}, Stock: salesStock{inventoryModule},
		Logger: log, TracerProvider: tp, MeterProvider: mp,
	})
	require.NoError(t, err)
	handler, err := newHandler(handlerConfig{
		Log: log, TracerProvider: tp, DB: readyDB{},
		Verifier: tokens, Permissions: permissions, Modules: moduleService,
		Routes: []func(*http.ServeMux){
			identityModule.RegisterRoutes, catalogModule.RegisterRoutes, inventoryModule.RegisterRoutes, salesModule.RegisterRoutes,
			moduleRoutes(moduleService, log),
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
	signup := func(t *testing.T, email, template string) string {
		t.Helper()
		rec := do(t, http.MethodPost, "/api/v1/auth/signup",
			`{"tenant_name":"Negocio de prueba","nit":"900123456-8","name":"Ana Gómez","email":"`+email+
				`","password":"clave-segura-123","business_template":"`+template+`"}`, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var session struct {
			AccessToken string `json:"access_token"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
		return session.AccessToken
	}
	// createProduct returns the default variant of a new simple product.
	createProduct := func(t *testing.T, token, sku, unit, price string) (productID, variantID string) {
		t.Helper()
		rec := do(t, http.MethodPost, "/api/v1/products",
			`{"sku":"`+sku+`","name":"Producto `+sku+`","base_unit_id":"`+unitID(t, do, token, unit)+
				`","cost":"1","price":"`+price+`","tax_rate":"0.19","attributes":{}}`, token)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var p struct {
			ID       string
			Variants []struct{ ID string }
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
		return p.ID, p.Variants[0].ID
	}

	owner := signup(t, "ana@unidades.co", "ferreteria")
	rec := do(t, http.MethodPost, "/api/v1/locations", `{"name":"Mostrador"}`, owner)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var location struct{ ID string }
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &location))
	rec = do(t, http.MethodPost, "/api/v1/cash-sessions", `{"location_id":"`+location.ID+`","opening_float":"0"}`, owner)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	cableProduct, cable := createProduct(t, owner, "cab-12", "m", "1800")
	screwProduct, screw := createProduct(t, owner, "tor-14", "und", "500")
	roll, box := unitID(t, do, owner, "rollo"), unitID(t, do, owner, "caja")
	unitPath := func(product, variant, unit string) string {
		return "/api/v1/products/" + product + "/variants/" + variant + "/units/" + unit
	}
	stockOf := func(t *testing.T, variantID string) string {
		t.Helper()
		rec := do(t, http.MethodGet, "/api/v1/locations/"+location.ID+"/stock?variant_id="+variantID, "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var page struct {
			Items []struct{ Quantity string }
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		require.Len(t, page.Items, 1)
		return page.Items[0].Quantity
	}
	receive := func(t *testing.T, variant, unit, qty, cost string) *httptest.ResponseRecorder {
		t.Helper()
		return do(t, http.MethodPost, "/api/v1/inventory/receipts",
			`{"variant_id":"`+variant+`","location_id":"`+location.ID+`","quantity":"`+qty+`","unit_cost":"`+cost+`","unit_id":"`+unit+`"}`, owner)
	}
	sell := func(t *testing.T, line, cash string) *httptest.ResponseRecorder {
		t.Helper()
		return do(t, http.MethodPost, "/api/v1/sales",
			`{"location_id":"`+location.ID+`","lines":[`+line+`],"payments":[{"method":"cash","amount":"`+cash+`"}]}`, owner)
	}

	t.Run("la ferretería tiene el módulo y le da a cada variante sus unidades alternativas", func(t *testing.T) {
		rec := do(t, http.MethodPut, unitPath(cableProduct, cable, roll), `{"factor":"100","price":"150000","barcode":"7709000000001"}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var p struct {
			Variants []struct {
				Units []struct {
					Unit          struct{ ID, Code string }
					Factor, Price string
				}
			}
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
		require.Len(t, p.Variants[0].Units, 1)
		u := p.Variants[0].Units[0]
		require.Equal(t, roll, u.Unit.ID)
		require.Equal(t, "rollo", u.Unit.Code)
		require.Equal(t, "100", u.Factor)
		require.Equal(t, "150000.00", u.Price)

		rec = do(t, http.MethodPut, unitPath(screwProduct, screw, box), `{"factor":"12","price":"5500"}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		rec = do(t, http.MethodPut, unitPath(cableProduct, cable, unitID(t, do, owner, "m")), `{"factor":"1","price":"1800"}`, owner)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "alternate_unit_is_base")
	})

	t.Run("la búsqueda encuentra el producto por el código de barras de una unidad alternativa", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/products?q=7709000000001", "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), cableProduct)
	})

	t.Run("un código de barras de una unidad no se puede repetir en una variante", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products",
			`{"sku":"otro-1","name":"Otro","base_unit_id":"`+unitID(t, do, owner, "und")+
				`","barcode":"7709000000001","cost":"1","price":"10","tax_rate":"0.19","attributes":{}}`, owner)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "barcode_taken")
	})

	t.Run("recibir 1 rollo suma 100 m al costo por metro", func(t *testing.T) {
		rec := receive(t, cable, roll, "1", "150000")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"unit_cost":"1500`)
		require.Equal(t, "100", stockOf(t, cable))
	})

	t.Run("vender 2,5 m de una variante con stock de 1 rollo deja 97,5 m", func(t *testing.T) {
		rec := sell(t, `{"variant_id":"`+cable+`","quantity":"2.5"}`, "10000")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, "97.5", stockOf(t, cable))
	})

	t.Run("vender 3 unidades de una caja ×12 deja 9 unidades", func(t *testing.T) {
		rec := receive(t, screw, box, "1", "4800")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, "12", stockOf(t, screw))

		rec = sell(t, `{"variant_id":"`+screw+`","quantity":"3"}`, "2000")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, "9", stockOf(t, screw))
	})

	t.Run("vende medio rollo al precio del rollo, la línea guarda la unidad y anular devuelve los metros", func(t *testing.T) {
		rec := sell(t, `{"variant_id":"`+cable+`","quantity":"0.5","unit_id":"`+roll+`"}`, "100000")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var sale struct {
			ID    string
			Lines []struct {
				Quantity     string  `json:"quantity"`
				UnitCode     *string `json:"unit_code"`
				UnitFactor   string  `json:"unit_factor"`
				BaseQuantity string  `json:"base_quantity"`
				UnitPrice    string  `json:"unit_price"`
				Gross        string  `json:"gross"`
			}
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sale))
		l := sale.Lines[0]
		require.Equal(t, "0.5", l.Quantity)
		require.Equal(t, "rollo", *l.UnitCode)
		require.Equal(t, "100", l.UnitFactor)
		require.Equal(t, "50", l.BaseQuantity)
		require.Equal(t, "150000.00", l.UnitPrice)
		require.Equal(t, "75000.00", l.Gross)
		require.Equal(t, "47.5", stockOf(t, cable))

		rec = do(t, http.MethodGet, "/api/v1/sales/"+sale.ID, "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"unit_code":"rollo"`)

		rec = do(t, http.MethodPost, "/api/v1/sales/"+sale.ID+"/void", `{"reason":"Error de digitación"}`, owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, "97.5", stockOf(t, cable))
	})

	t.Run("rechaza vender en una unidad que la variante no tiene", func(t *testing.T) {
		rec := sell(t, `{"variant_id":"`+screw+`","quantity":"1","unit_id":"`+roll+`"}`, "200000")
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "unit_not_available")
		require.Equal(t, "9", stockOf(t, screw))
	})

	t.Run("quitar la unidad deja de venderla", func(t *testing.T) {
		rec := do(t, http.MethodDelete, unitPath(screwProduct, screw, box), "", owner)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"units":[]`)

		rec = sell(t, `{"variant_id":"`+screw+`","quantity":"1","unit_id":"`+box+`"}`, "10000")
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "unit_not_available")
	})

	t.Run("sin el módulo activo las rutas de unidades alternativas responden 404", func(t *testing.T) {
		clothes := signup(t, "luz@ropa.co", "ropa_calzado")
		product, variant := createProduct(t, clothes, "cam-1", "und", "30000")
		rec := do(t, http.MethodGet, "/api/v1/units", "", clothes)
		var units struct {
			Items []struct{ ID, Code string } `json:"items"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &units))
		var other string
		for _, u := range units.Items {
			if u.Code != "und" {
				other = u.ID
				break
			}
		}
		require.NotEmpty(t, other)

		rec = do(t, http.MethodPut, unitPath(product, variant, other), `{"factor":"3","price":"80000"}`, clothes)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"not_found"`)
	})

	t.Run("una empresa no ve ni usa las unidades alternativas de otra", func(t *testing.T) {
		other := signup(t, "leo@otra.co", "ferreteria")
		rec := do(t, http.MethodPut, unitPath(cableProduct, cable, roll), `{"factor":"50","price":"1"}`, other)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

		rec = do(t, http.MethodGet, "/api/v1/products?q=7709000000001", "", other)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.NotContains(t, rec.Body.String(), cableProduct)
	})
}
