//go:build integration

package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
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

// TestSalesFlow runs sales through the whole stack: the catalog's prices,
// the inventory's stock exits in the sale's transaction, the consecutive
// number, voiding and the cashier's cash session up to its closing, with the
// permissions of each role.
func TestSalesFlow(t *testing.T) {
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
	salesModule, err := sales.New(sales.Config{
		DB: database, Authorize: identity.Authorize,
		Products: salesCatalog{catalogModule}, Stock: salesStock{inventoryModule},
		Logger: log, TracerProvider: tp, MeterProvider: mp,
	})
	require.NoError(t, err)
	handler, err := newHandler(handlerConfig{
		Log: log, TracerProvider: tp, DB: readyDB{},
		Verifier: tokens, Authorize: identity.Authorize,
		Routes: []func(*http.ServeMux){
			identityModule.RegisterRoutes, catalogModule.RegisterRoutes, inventoryModule.RegisterRoutes, salesModule.RegisterRoutes,
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
	id := func(t *testing.T, rec *httptest.ResponseRecorder) string {
		t.Helper()
		var body struct{ ID string }
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		return body.ID
	}
	signup := func(t *testing.T, email string) (token string, tenantID uuid.UUID) {
		t.Helper()
		rec := do(t, http.MethodPost, "/api/v1/auth/signup",
			`{"tenant_name":"Ferretería El Tornillo","nit":"890903938-8","name":"Ana Gómez","email":"`+email+`","password":"clave-segura-123"}`, "")
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var session struct {
			AccessToken string `json:"access_token"`
			User        struct {
				TenantID uuid.UUID `json:"tenant_id"`
			} `json:"user"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
		return session.AccessToken, session.User.TenantID
	}

	ownerToken, tenantID := signup(t, "ana@ventas.co")
	cashierToken, _, err := tokens.Issue(tenantID, uuid.Must(uuid.NewV7()), "cashier")
	require.NoError(t, err)
	warehouseToken, _, err := tokens.Issue(tenantID, uuid.Must(uuid.NewV7()), "warehouse")
	require.NoError(t, err)

	rec := do(t, http.MethodPost, "/api/v1/products",
		`{"sku":"cem-50","name":"Cemento gris 50kg","base_unit":"bulto","cost":"25000","price":"30000","tax_rate":"0.19"}`, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	productID := id(t, rec)
	rec = do(t, http.MethodPost, "/api/v1/locations", `{"name":"Mostrador"}`, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	locationID := id(t, rec)
	rec = do(t, http.MethodPost, "/api/v1/inventory/receipts",
		`{"product_id":"`+productID+`","location_id":"`+locationID+`","quantity":"10","unit_cost":"25000"}`, ownerToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	stock := func(t *testing.T) string {
		t.Helper()
		rec := do(t, http.MethodGet, "/api/v1/locations/"+locationID+"/stock?product_id="+productID, "", ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var page struct {
			Items []struct{ Quantity string }
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		require.Len(t, page.Items, 1)
		return page.Items[0].Quantity
	}
	saleBody := func(qty, cash string) string {
		return `{"location_id":"` + locationID + `","lines":[{"product_id":"` + productID + `","quantity":"` + qty + `"}],` +
			`"payments":[{"method":"cash","amount":"` + cash + `"}]}`
	}

	var saleID, sessionID string

	t.Run("sin caja abierta el cajero no puede vender", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/cash-sessions/current", "", cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"required":true}`, rec.Body.String())

		rec = do(t, http.MethodPost, "/api/v1/sales", saleBody("1", "35700"), cashierToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "cash_session_required")
		require.Equal(t, "10", stock(t))
	})

	t.Run("el cajero abre su caja con base; no puede abrir dos ni en una sede inexistente", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/cash-sessions", `{"location_id":"`+uuid.Must(uuid.NewV7()).String()+`","opening_float":"100000"}`, cashierToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "location_not_found")

		rec = do(t, http.MethodPost, "/api/v1/cash-sessions", `{"location_id":"`+locationID+`","opening_float":"100000"}`, cashierToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		sessionID = id(t, rec)
		require.Contains(t, rec.Body.String(), `"status":"open"`)

		rec = do(t, http.MethodPost, "/api/v1/cash-sessions", `{"location_id":"`+locationID+`","opening_float":"0"}`, cashierToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "cash_session_already_open")

		rec = do(t, http.MethodGet, "/api/v1/cash-sessions/current", "", cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), sessionID)
		require.NotContains(t, rec.Body.String(), "expected", "arqueo ciego")
	})

	t.Run("el cajero confirma una venta: descuenta el stock y responde el consecutivo", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/sales", saleBody("2.5", "100000"), cashierToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		var sale struct {
			ID     string
			Number int64
			Status string
			Totals struct {
				Total string
				Taxes []struct{ Rate, Base, Tax string }
			}
			Change string
			Lines  []struct {
				ProductName string `json:"product_name"`
				UnitPrice   string `json:"unit_price"`
			}
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sale))
		saleID = sale.ID
		require.Contains(t, rec.Body.String(), `"cash_session_id":"`+sessionID+`"`)
		require.Equal(t, int64(1), sale.Number)
		require.Equal(t, "confirmed", sale.Status)
		// 2.5 × 30000 = 75000 + IVA 14250 = 89250
		require.Equal(t, "89250.00", sale.Totals.Total)
		require.Equal(t, "10750.00", sale.Change)
		require.Equal(t, "Cemento gris 50kg", sale.Lines[0].ProductName)
		require.Equal(t, "30000.00", sale.Lines[0].UnitPrice)
		require.Len(t, sale.Totals.Taxes, 1)
		require.Equal(t, "0.19", sale.Totals.Taxes[0].Rate)
		require.Equal(t, "7.5", stock(t))
	})

	t.Run("si no hay stock suficiente responde 409 con el producto y no guarda la venta", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/sales", saleBody("8", "300000"), cashierToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		var body struct {
			Error struct {
				Code, Message string
				Details       map[string]string
			}
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, "insufficient_stock", body.Error.Code)
		require.Equal(t, "No hay stock suficiente de Cemento gris 50kg.", body.Error.Message)
		require.Equal(t, productID, body.Error.Details["product_id"])
		require.Equal(t, "7.5", stock(t))

		next := do(t, http.MethodPost, "/api/v1/sales", saleBody("1", "35700"), cashierToken)
		require.Equal(t, http.StatusCreated, next.Code, next.Body.String())
		require.Contains(t, next.Body.String(), `"number":2`, "la venta fallida no dejó hueco")
	})

	t.Run("el cajero no puede descontar sin tope configurado", func(t *testing.T) {
		body := `{"location_id":"` + locationID + `","lines":[{"product_id":"` + productID + `","quantity":"1","discount":"1000"}],` +
			`"payments":[{"method":"cash","amount":"35700"}]}`
		rec := do(t, http.MethodPost, "/api/v1/sales", body, cashierToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "discount_exceeds_limit")
	})

	t.Run("rechaza pago insuficiente y una sede distinta a la de la caja con 422", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/sales", saleBody("1", "100"), cashierToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "insufficient_payment")

		body := strings.Replace(saleBody("1", "35700"), locationID, uuid.Must(uuid.NewV7()).String(), 1)
		rec = do(t, http.MethodPost, "/api/v1/sales", body, cashierToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "cash_session_location_mismatch")
	})

	t.Run("el bodeguero no puede vender ni ver ventas", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/sales", saleBody("1", "35700"), warehouseToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		rec = do(t, http.MethodGet, "/api/v1/sales", "", warehouseToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})

	t.Run("dos ventas concurrentes no generan el mismo consecutivo", func(t *testing.T) {
		var wg sync.WaitGroup
		numbers := make([]int64, 4)
		for i := range numbers {
			wg.Go(func() {
				rec := do(t, http.MethodPost, "/api/v1/sales", saleBody("0.5", "17850"), cashierToken)
				if rec.Code != http.StatusCreated {
					return
				}
				var sale struct{ Number int64 }
				_ = json.Unmarshal(rec.Body.Bytes(), &sale)
				numbers[i] = sale.Number
			})
		}
		wg.Wait()
		require.ElementsMatch(t, []int64{3, 4, 5, 6}, numbers)
		require.Equal(t, "4.5", stock(t))
	})

	t.Run("el cajero ve el detalle y las ventas del día", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/sales/"+saleID, "", cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"number":1`)

		rec = do(t, http.MethodGet, "/api/v1/sales?location_id="+locationID, "", cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var page struct {
			Items []struct{ Number int64 }
			Total int
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
		require.Equal(t, 6, page.Total)
		// Las ventas concurrentes pueden tener la hora en otro orden que el
		// consecutivo; la primera venta sí es la más antigua.
		require.Equal(t, int64(1), page.Items[len(page.Items)-1].Number, "la más reciente primero")
	})

	t.Run("un cajero no puede anular", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/sales/"+saleID+"/void", `{"reason":"Error"}`, cashierToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})

	t.Run("la anulación restaura el stock exacto y no se repite", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/sales/"+saleID+"/void", `{"reason":"El cliente desistió"}`, ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"status":"voided"`)
		require.Contains(t, rec.Body.String(), `"reason":"El cliente desistió"`)
		require.Equal(t, "7", stock(t))

		rec = do(t, http.MethodPost, "/api/v1/sales/"+saleID+"/void", `{"reason":"Otra vez"}`, ownerToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "sale_not_confirmed")
		require.Equal(t, "7", stock(t))

		rec = do(t, http.MethodGet, "/api/v1/products/"+productID+"/kardex", "", ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"type":"sale_void"`)
	})

	t.Run("el cajero registra un egreso y cierra la caja con arqueo", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/cash-sessions/"+sessionID+"/movements",
			`{"kind":"out","amount":"7100","reason":"Pago del domicilio"}`, cashierToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		rec = do(t, http.MethodPost, "/api/v1/cash-sessions/"+sessionID+"/close",
			`{"counts":[{"denomination":"100000","quantity":1}]}`, ownerToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "cash_session_not_owner")

		// The voided first sale does not count: 35700 + 4 × 17850 = 107100 in
		// cash, exact. 100000 + 107100 − 7100 = 200000.
		rec = do(t, http.MethodPost, "/api/v1/cash-sessions/"+sessionID+"/close",
			`{"counts":[{"denomination":"100000","quantity":1},{"denomination":"50000","quantity":1},{"denomination":"20000","quantity":2},{"denomination":"5000","quantity":1}],"notes":"Turno de la mañana"}`,
			cashierToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var session struct {
			Status  string
			Closing struct {
				SalesCount   int    `json:"sales_count"`
				CashReceived string `json:"cash_received"`
				CashOut      string `json:"cash_out"`
				Expected     string
				Counted      string
				Difference   string
				Result       string
			}
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
		require.Equal(t, "closed", session.Status)
		require.Equal(t, 5, session.Closing.SalesCount)
		require.Equal(t, "107100.00", session.Closing.CashReceived)
		require.Equal(t, "7100.00", session.Closing.CashOut)
		require.Equal(t, "200000.00", session.Closing.Expected)
		require.Equal(t, "195000.00", session.Closing.Counted)
		require.Equal(t, "-5000.00", session.Closing.Difference)
		require.Equal(t, "shortage", session.Closing.Result)
	})

	t.Run("el cierre queda inmutable y sin caja no se vende", func(t *testing.T) {
		rec := do(t, http.MethodPost, "/api/v1/cash-sessions/"+sessionID+"/close", `{"counts":[]}`, cashierToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "cash_session_closed")
		rec = do(t, http.MethodPost, "/api/v1/cash-sessions/"+sessionID+"/movements", `{"kind":"in","amount":"1000","reason":"Base"}`, cashierToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		rec = do(t, http.MethodPost, "/api/v1/sales", saleBody("1", "35700"), cashierToken)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "cash_session_required")

		rec = do(t, http.MethodGet, "/api/v1/cash-sessions/"+sessionID, "", ownerToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"expected":"200000.00"`)
		rec = do(t, http.MethodGet, "/api/v1/cash-sessions/"+sessionID, "", warehouseToken)
		require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})

	t.Run("aislamiento: otro tenant no ve ni anula la venta", func(t *testing.T) {
		otherToken, _ := signup(t, "otro@ventas.co")

		rec := do(t, http.MethodGet, "/api/v1/cash-sessions/"+sessionID, "", otherToken)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "cash_session_not_found")
		rec = do(t, http.MethodPost, "/api/v1/locations", `{"name":"Bodega"}`, otherToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		otherLocation := id(t, rec)
		rec = do(t, http.MethodPost, "/api/v1/cash-sessions", `{"location_id":"`+otherLocation+`","opening_float":"0"}`, otherToken)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

		rec = do(t, http.MethodGet, "/api/v1/sales/"+saleID, "", otherToken)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "sale_not_found")

		rec = do(t, http.MethodPost, "/api/v1/sales/"+saleID+"/void", `{"reason":"Ajena"}`, otherToken)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

		rec = do(t, http.MethodGet, "/api/v1/sales", "", otherToken)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"total":0`)

		rec = do(t, http.MethodPost, "/api/v1/sales", strings.Replace(saleBody("1", "35700"), locationID, otherLocation, 1), otherToken)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "product_not_found", "no puede vender productos de otro tenant")
	})
}
