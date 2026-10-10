package httpadapter_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

var now = time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// stubUseCases records the last call and answers with what each test sets.
type stubUseCases struct {
	called       bool
	actor        app.Actor
	name         string
	receipt      app.ReceiptInput
	adjustment   app.AdjustmentInput
	transfer     app.TransferInput
	stockFilter  domain.StockFilter
	kardexFilter domain.KardexFilter

	location  *domain.Location
	locations []*domain.Location
	movement  *domain.StockMovement
	out, in   *domain.StockMovement
	levels    domain.StockLevelPage
	kardex    domain.MovementPage
	err       error
}

func (s *stubUseCases) CreateLocation(_ context.Context, a app.Actor, name string) (*domain.Location, error) {
	s.called, s.actor, s.name = true, a, name
	return s.location, s.err
}

func (s *stubUseCases) ListLocations(_ context.Context, a app.Actor) ([]*domain.Location, error) {
	s.called, s.actor = true, a
	return s.locations, s.err
}

func (s *stubUseCases) ReceiveStock(_ context.Context, a app.Actor, in app.ReceiptInput) (*domain.StockMovement, error) {
	s.called, s.actor, s.receipt = true, a, in
	return s.movement, s.err
}

func (s *stubUseCases) AdjustStock(_ context.Context, a app.Actor, in app.AdjustmentInput) (*domain.StockMovement, error) {
	s.called, s.actor, s.adjustment = true, a, in
	return s.movement, s.err
}

func (s *stubUseCases) TransferStock(_ context.Context, a app.Actor, in app.TransferInput) (*domain.StockMovement, *domain.StockMovement, error) {
	s.called, s.actor, s.transfer = true, a, in
	return s.out, s.in, s.err
}

func (s *stubUseCases) ListStock(_ context.Context, a app.Actor, f domain.StockFilter) (domain.StockLevelPage, error) {
	s.called, s.actor, s.stockFilter = true, a, f
	return s.levels, s.err
}

func (s *stubUseCases) GetKardex(_ context.Context, a app.Actor, f domain.KardexFilter) (domain.MovementPage, error) {
	s.called, s.actor, s.kardexFilter = true, a, f
	return s.kardex, s.err
}

// seeCostOnlyOwner grants catalog:write only to owners, like a reduced
// permission matrix.
func seeCostOnlyOwner(role, permission string) bool {
	return role == "owner" || permission != "catalog:write"
}

var owner = auth.Principal{TenantID: newID(), UserID: newID(), Role: "owner"}

func warehouse() auth.Principal {
	p := owner
	p.Role = "warehouse"
	return p
}

func serve(t *testing.T, uc *stubUseCases, p *auth.Principal, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, seeCostOnlyOwner, slog.New(slog.DiscardHandler))
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

func requireError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, code, body.Error.Code)
	require.NotEmpty(t, body.Error.Message)
}

func decodeMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &m), rec.Body.String())
	return m
}

// receipt builds the movement of an entry of 2.5 units at 1200.5 over 10 at 1000.
func receipt(t *testing.T) *domain.StockMovement {
	t.Helper()
	level, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
		TenantID: owner.TenantID, ProductID: newID(), LocationID: newID(), Quantity: dec("10"), AverageCost: dec("1000"),
	})
	require.NoError(t, err)
	m, err := level.Receive(dec("2.5"), dec("1200.5"), domain.MovementInfo{UserID: owner.UserID, OccurredAt: now})
	require.NoError(t, err)
	return m
}

func TestReceiveStock(t *testing.T) {
	productID, locationID := newID(), newID()
	body := `{"product_id":"` + productID.String() + `","location_id":"` + locationID.String() + `","quantity":"2.5","unit_cost":"1200.5"}`

	t.Run("registrar una entrada devuelve 201 con cantidades y costos como string", func(t *testing.T) {
		m := receipt(t)
		uc := &stubUseCases{movement: m}
		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/inventory/receipts", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		got := decodeMap(t, rec)
		require.Equal(t, m.ID().String(), got["id"])
		require.Equal(t, "purchase_in", got["type"])
		require.Equal(t, "2.5", got["quantity"])
		require.Equal(t, "12.5", got["balance_after"])
		require.Equal(t, "1200.50", got["unit_cost"])
		require.Equal(t, "1040.10", got["average_cost_after"])
		require.Equal(t, owner.UserID.String(), got["user_id"])
		require.NotContains(t, got, "reason")
		require.NotContains(t, got, "reference")
	})

	t.Run("pasa los datos al caso de uso y el tenant sale del token, no del body", func(t *testing.T) {
		uc := &stubUseCases{movement: receipt(t)}
		withTenant := strings.Replace(body, "{", `{"tenant_id":"`+newID().String()+`",`, 1)
		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/inventory/receipts", withTenant)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, app.Actor{TenantID: owner.TenantID, UserID: owner.UserID}, uc.actor)
		require.Equal(t, productID, uc.receipt.ProductID)
		require.Equal(t, locationID, uc.receipt.LocationID)
		require.True(t, dec("2.5").Equal(uc.receipt.Quantity))
		require.True(t, dec("1200.5").Equal(uc.receipt.UnitCost))
	})

	t.Run("quien no puede editar el catálogo no ve los costos", func(t *testing.T) {
		p := warehouse()
		rec := serve(t, &stubUseCases{movement: receipt(t)}, &p, http.MethodPost, "/api/v1/inventory/receipts", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		got := decodeMap(t, rec)
		require.NotContains(t, got, "unit_cost")
		require.NotContains(t, got, "average_cost_after")
	})

	t.Run("sede inactiva responde 422 location_inactive", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrLocationInactive}, &owner, http.MethodPost, "/api/v1/inventory/receipts", body)
		requireError(t, rec, http.StatusUnprocessableEntity, "location_inactive")
	})

	t.Run("producto inexistente responde 422 product_not_found", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrProductNotFound}, &owner, http.MethodPost, "/api/v1/inventory/receipts", body)
		requireError(t, rec, http.StatusUnprocessableEntity, "product_not_found")
	})

	t.Run("sin principal responde 401 y no llama al caso de uso", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, nil, http.MethodPost, "/api/v1/inventory/receipts", body)
		requireError(t, rec, http.StatusUnauthorized, "unauthorized")
		require.False(t, uc.called)
	})
}

func TestAdjustStock(t *testing.T) {
	productID, locationID := newID(), newID()
	body := `{"product_id":"` + productID.String() + `","location_id":"` + locationID.String() + `","quantity":"-3","reason":"Bulto roto"}`

	t.Run("pasa la cantidad con signo y el motivo", func(t *testing.T) {
		level, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
			TenantID: owner.TenantID, ProductID: productID, LocationID: locationID, Quantity: dec("5"), AverageCost: dec("100"),
		})
		require.NoError(t, err)
		m, err := level.Adjust(dec("-3"), "Bulto roto", domain.StockPolicy{}, domain.MovementInfo{UserID: owner.UserID, OccurredAt: now})
		require.NoError(t, err)
		uc := &stubUseCases{movement: m}

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/inventory/adjustments", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.True(t, dec("-3").Equal(uc.adjustment.Quantity))
		require.Equal(t, "Bulto roto", uc.adjustment.Reason)
		got := decodeMap(t, rec)
		require.Equal(t, "-3", got["quantity"])
		require.Equal(t, "Bulto roto", got["reason"])
	})

	t.Run("insufficient_stock responde 422 con el código", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrInsufficientStock}, &owner, http.MethodPost, "/api/v1/inventory/adjustments", body)
		requireError(t, rec, http.StatusUnprocessableEntity, "insufficient_stock")
	})

	t.Run("ajuste sin motivo responde 422 adjustment_reason_required", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrAdjustmentReasonRequired}, &owner, http.MethodPost, "/api/v1/inventory/adjustments", body)
		requireError(t, rec, http.StatusUnprocessableEntity, "adjustment_reason_required")
	})
}

func TestTransferStock(t *testing.T) {
	productID, from, to := newID(), newID(), newID()
	body := `{"product_id":"` + productID.String() + `","from_location_id":"` + from.String() + `","to_location_id":"` + to.String() + `","quantity":"4"}`

	t.Run("devuelve 201 con los dos movimientos y su referencia", func(t *testing.T) {
		origin, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
			TenantID: owner.TenantID, ProductID: productID, LocationID: from, Quantity: dec("10"), AverageCost: dec("100"),
		})
		require.NoError(t, err)
		dest, err := domain.NewStockLevel(owner.TenantID, productID, to)
		require.NoError(t, err)
		out, in, err := domain.Transfer(origin, dest, dec("4"), domain.StockPolicy{}, owner.UserID, now)
		require.NoError(t, err)
		uc := &stubUseCases{out: out, in: in}

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/inventory/transfers", body)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, app.TransferInput{ProductID: productID, FromLocationID: from, ToLocationID: to, Quantity: uc.transfer.Quantity}, uc.transfer)
		require.True(t, dec("4").Equal(uc.transfer.Quantity))
		var got struct {
			Out, In struct {
				Type, Quantity string
				Reference      struct{ Kind, ID string } `json:"reference"`
			}
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Equal(t, "transfer_out", got.Out.Type)
		require.Equal(t, "-4", got.Out.Quantity)
		require.Equal(t, "transfer_in", got.In.Type)
		require.Equal(t, "transfer", got.Out.Reference.Kind)
		require.Equal(t, out.Reference().ID.String(), got.In.Reference.ID)
	})

	t.Run("traslado a la misma sede responde 422 same_location_transfer", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrSameLocationTransfer}, &owner, http.MethodPost, "/api/v1/inventory/transfers", body)
		requireError(t, rec, http.StatusUnprocessableEntity, "same_location_transfer")
	})
}

func TestLocations(t *testing.T) {
	t.Run("crea una sede", func(t *testing.T) {
		loc, err := domain.NewLocation(owner.TenantID, "Bodega principal", now)
		require.NoError(t, err)
		uc := &stubUseCases{location: loc}

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/locations", `{"name":"Bodega principal"}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, "Bodega principal", uc.name)
		got := decodeMap(t, rec)
		require.Equal(t, loc.ID().String(), got["id"])
		require.Equal(t, true, got["active"])
	})

	t.Run("nombre repetido responde 409 location_name_taken", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrLocationNameTaken}, &owner, http.MethodPost, "/api/v1/locations", `{"name":"Bodega"}`)
		requireError(t, rec, http.StatusConflict, "location_name_taken")
	})

	t.Run("lista las sedes", func(t *testing.T) {
		loc, err := domain.NewLocation(owner.TenantID, "Bodega", now)
		require.NoError(t, err)
		rec := serve(t, &stubUseCases{locations: []*domain.Location{loc}}, &owner, http.MethodGet, "/api/v1/locations", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Len(t, decodeMap(t, rec)["items"], 1)
	})
}

func TestListStock(t *testing.T) {
	locationID := newID()

	t.Run("lista las existencias con la paginación por defecto y oculta el costo a quien no edita el catálogo", func(t *testing.T) {
		level, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
			TenantID: owner.TenantID, ProductID: newID(), LocationID: locationID, Quantity: dec("7.5"), AverageCost: dec("100"),
		})
		require.NoError(t, err)
		uc := &stubUseCases{levels: domain.StockLevelPage{Items: []*domain.StockLevel{level}, Total: 1}}
		p := warehouse()

		rec := serve(t, uc, &p, http.MethodGet, "/api/v1/locations/"+locationID.String()+"/stock", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, domain.StockFilter{LocationID: locationID, Limit: 20}, uc.stockFilter)
		require.Contains(t, rec.Body.String(), `"quantity":"7.5"`)
		require.NotContains(t, rec.Body.String(), "average_cost")
	})

	t.Run("el dueño ve el costo promedio", func(t *testing.T) {
		level, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
			TenantID: owner.TenantID, ProductID: newID(), LocationID: locationID, Quantity: dec("1"), AverageCost: dec("100"),
		})
		require.NoError(t, err)
		uc := &stubUseCases{levels: domain.StockLevelPage{Items: []*domain.StockLevel{level}, Total: 1}}

		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/locations/"+locationID.String()+"/stock", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"average_cost":"100.00"`)
	})

	t.Run("pasa producto y paginación", func(t *testing.T) {
		productID := newID()
		uc := &stubUseCases{}
		rec := serve(t, uc, &owner, http.MethodGet,
			"/api/v1/locations/"+locationID.String()+"/stock?product_id="+productID.String()+"&limit=5&offset=10", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, domain.StockFilter{LocationID: locationID, ProductID: &productID, Limit: 5, Offset: 10}, uc.stockFilter)
	})

	t.Run("sede inexistente responde 404 location_not_found", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrLocationNotFound}, &owner, http.MethodGet, "/api/v1/locations/"+locationID.String()+"/stock", "")
		requireError(t, rec, http.StatusNotFound, "location_not_found")
	})
}

func TestGetKardex(t *testing.T) {
	t.Run("pasa producto, sede y paginación y devuelve la página", func(t *testing.T) {
		productID, locationID := newID(), newID()
		m := receipt(t)
		uc := &stubUseCases{kardex: domain.MovementPage{Items: []*domain.StockMovement{m}, Total: 3}}

		rec := serve(t, uc, &owner, http.MethodGet,
			"/api/v1/products/"+productID.String()+"/kardex?location_id="+locationID.String()+"&offset=2", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, domain.KardexFilter{ProductID: productID, LocationID: &locationID, Limit: 20, Offset: 2}, uc.kardexFilter)
		got := decodeMap(t, rec)
		require.EqualValues(t, 3, got["total"])
		require.Len(t, got["items"], 1)
	})
}
