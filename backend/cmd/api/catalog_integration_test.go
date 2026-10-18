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
	moduleService, permissions := testModules(t, database)
	tokens, err := auth.NewJWT(testSecret, 15*time.Minute)
	require.NoError(t, err)
	log := slog.New(slog.DiscardHandler)
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
		Routes: []func(*http.ServeMux){identityModule.RegisterRoutes, catalogModule.RegisterRoutes, moduleRoutes(moduleService, log)},
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
		`{"tenant_name":"Ferretería El Tornillo","nit":"900123456-8","name":"Ana Gómez","email":"ana@catalogo.co","password":"clave-segura-123","business_template":"ferreteria"}`, "")
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

	t.Run("un tenant nuevo de ferretería queda con las unidades de su plantilla", func(t *testing.T) {
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

	body := `{"sku":"tub-pvc-12","name":"Tubo PVC presión 1/2 pulgada","base_unit_id":"` + unitID("m") +
		`","cost":"8333.33","price":"10000","tax_rate":"0.19","attributes":{}}`
	var productID string

	t.Run("el dueño crea un producto y recibe el dinero como string", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products", body, ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var p map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
		variant := p["variants"].([]any)[0].(map[string]any)
		require.Equal(t, "TUB-PVC-12", variant["sku"])
		require.Equal(t, "10000.00", variant["price"])
		require.Equal(t, "8333.33", variant["cost"])
		require.NotContains(t, p, "sku", "the default variant's fields are only in variants")
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

	t.Run("el producto creado lleva su unidad", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/products/"+productID, "", ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var p map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
		require.Equal(t, unitID("m"), p["base_unit_id"])
		require.Equal(t, map[string]any{"id": unitID("m"), "code": "m", "name": "Metro"}, p["unit"])
	})

	t.Run("crea un producto con base_unit_id y lo devuelve con su unidad", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products",
			`{"sku":"cem-50","name":"Cemento gris 50 kg","base_unit_id":"`+unitID("bulto")+`","cost":"25000","price":"30000","tax_rate":"0.19","attributes":{}}`, ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var p map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
		require.Equal(t, unitID("bulto"), p["base_unit_id"])
		require.Equal(t, map[string]any{"id": unitID("bulto"), "code": "bulto", "name": "Bulto"}, p["unit"])
	})

	t.Run("un producto sin unidad, sin atributos o con el código de unidad responde 400 por el contrato", func(t *testing.T) {
		for _, body := range []string{
			`{"sku":"cam-s","name":"Camiseta básica","cost":"12000","price":"25000","tax_rate":"0.19","attributes":{}}`,
			`{"sku":"cam-s","name":"Camiseta básica","base_unit_id":"` + unitID("und") + `","cost":"12000","price":"25000","tax_rate":"0.19"}`,
			`{"sku":"cam-s","name":"Camiseta básica","base_unit":"und","cost":"12000","price":"25000","tax_rate":"0.19","attributes":{}}`,
		} {
			rec := do(t, http.MethodPost, "/api/v1/products", body, ownerToken)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	t.Run("un producto con una unidad de otro tenant responde 422 unit_not_found", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products",
			`{"sku":"cam-s","name":"Camiseta básica","base_unit_id":"`+uuid.Must(uuid.NewV7()).String()+`","cost":"12000","price":"25000","tax_rate":"0.19","attributes":{}}`, ownerToken)
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

	var attributeID string

	t.Run("el dueño define atributos y el cajero los lista", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/product-attributes",
			`{"key":"material_tubo","label":"Material del tubo","type":"list","options":["PVC","Cobre"],"required":false}`, ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var d struct {
			ID      string   `json:"id"`
			Key     string   `json:"key"`
			Type    string   `json:"type"`
			Options []string `json:"options"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &d))
		require.Equal(t, "material_tubo", d.Key)
		require.Equal(t, "list", d.Type)
		require.Equal(t, []string{"PVC", "Cobre"}, d.Options)
		attributeID = d.ID

		rec = do(t, http.MethodPost, "/api/v1/product-attributes",
			`{"key":"importado","label":"Importado","type":"boolean","required":false}`, ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		rec = do(t, http.MethodGet, "/api/v1/product-attributes", "", cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"material_tubo"`)
		require.Contains(t, rec.Body.String(), `"importado"`)
	})

	t.Run("el cajero no define atributos", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/product-attributes",
			`{"key":"marca","label":"Marca","type":"text","required":false}`, cashierToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})

	t.Run("una clave repetida responde 409 attribute_key_taken y un tipo desconocido 400", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/product-attributes",
			`{"key":"material_tubo","label":"Otro","type":"text","required":false}`, ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "attribute_key_taken")

		rec = do(t, http.MethodPost, "/api/v1/product-attributes",
			`{"key":"vence","label":"Vence","type":"date","required":false}`, ownerToken)
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})

	t.Run("crea un producto con atributos y los devuelve normalizados", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products",
			`{"sku":"tub-cobre","name":"Tubo de cobre","base_unit_id":"`+unitID("m")+`","cost":"1000","price":"2000","tax_rate":"0.19",`+
				`"attributes":{"material_tubo":"cobre","importado":true}}`, ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var p struct {
			ID         string         `json:"id"`
			Attributes map[string]any `json:"attributes"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
		require.Equal(t, map[string]any{"material_tubo": "Cobre", "importado": true}, p.Attributes)

		rec = do(t, http.MethodGet, "/api/v1/products/"+p.ID, "", cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"attributes":{`)
	})

	t.Run("un producto sin atributos los devuelve vacíos", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/products/"+productID, "", ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"attributes":{}`)
	})

	t.Run("una opción no definida responde 422 invalid_attribute_value con el atributo", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products",
			`{"sku":"tub-hierro","name":"Tubo de hierro","base_unit_id":"`+unitID("m")+`","cost":"1000","price":"2000","tax_rate":"0.19",`+
				`"attributes":{"material_tubo":"Hierro"}}`, ownerToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		var body struct {
			Error struct {
				Code    string            `json:"code"`
				Message string            `json:"message"`
				Details map[string]string `json:"details"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, "invalid_attribute_value", body.Error.Code)
		require.Equal(t, map[string]string{"attribute": "material_tubo"}, body.Error.Details)
		require.Contains(t, body.Error.Message, "Material")
	})

	t.Run("un atributo no definido responde 422 unknown_attribute y un valor numérico JSON 400", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/products",
			`{"sku":"tub-x","name":"Tubo","base_unit_id":"`+unitID("m")+`","cost":"1000","price":"2000","tax_rate":"0.19",`+
				`"attributes":{"color":"Rojo"}}`, ownerToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "unknown_attribute")

		rec = do(t, http.MethodPost, "/api/v1/products",
			`{"sku":"tub-x","name":"Tubo","base_unit_id":"`+unitID("m")+`","cost":"1000","price":"2000","tax_rate":"0.19",`+
				`"attributes":{"material_tubo":12}}`, ownerToken)
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})

	t.Run("borrar un atributo en uso responde 409 attribute_in_use", func(t *testing.T) {
		rec := do(t, http.MethodDelete, "/api/v1/product-attributes/"+attributeID, "", ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "attribute_in_use")

		rec = do(t, http.MethodPut, "/api/v1/product-attributes/"+attributeID,
			`{"label":"Material del tubo","options":["PVC"],"required":false}`, ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "attribute_option_in_use")

		rec = do(t, http.MethodDelete, "/api/v1/product-attributes/"+uuid.Must(uuid.NewV7()).String(), "", ownerToken)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "attribute_not_found")
	})

	t.Run("las variantes usan los ejes del tenant", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/variant-axes", `{"name":"Color","values":["Rojo","Azul"]}`, ownerToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var axis struct {
			ID     string   `json:"id"`
			Name   string   `json:"name"`
			Values []string `json:"values"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &axis))
		require.Equal(t, "color", axis.Name)
		require.Equal(t, []string{"Rojo", "Azul"}, axis.Values)

		rec = do(t, http.MethodGet, "/api/v1/products/"+productID, "", ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var p struct {
			Variants []struct {
				ID string `json:"id"`
			} `json:"variants"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
		variantPath := "/api/v1/products/" + productID + "/variants"

		rec = do(t, http.MethodPut, variantPath+"/"+p.Variants[0].ID,
			`{"sku":"tub-pvc-12","cost":"8333.33","price":"10000","axis_values":{"color":"rojo"},"active":true}`, ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"axis_values":{"color":"Rojo"}`)

		// Adding variants is the variants module's.
		rec = do(t, http.MethodPut, "/api/v1/modules/variants", `{"active":true}`, ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPost, variantPath, `{"sku":"tub-pvc-12-v","cost":"1","price":"2","axis_values":{"color":"Verde"}}`, ownerToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "axis_value_not_allowed")
		require.Contains(t, rec.Body.String(), `"axis":"color"`)

		rec = do(t, http.MethodPost, variantPath, `{"sku":"tub-pvc-12-v","cost":"1","price":"2","axis_values":{"talla":"M"}}`, ownerToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "unknown_axis")

		rec = do(t, http.MethodPut, "/api/v1/variant-axes/"+axis.ID, `{"values":["Azul"]}`, ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "axis_value_in_use")

		rec = do(t, http.MethodDelete, "/api/v1/variant-axes/"+axis.ID, "", ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "axis_in_use")

		rec = do(t, http.MethodGet, "/api/v1/variant-axes", "", cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"color"`)
	})
}
