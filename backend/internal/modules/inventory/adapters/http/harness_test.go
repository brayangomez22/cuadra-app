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
	called      bool
	actor       app.Actor
	name        string
	receipt     app.ReceiptInput
	adjustment  app.AdjustmentInput
	transfer    app.TransferInput
	stockQuery  app.StockQuery
	kardexQuery app.KardexQuery

	location  *domain.Location
	locations []*domain.Location
	movement  *domain.StockMovement
	out, in   *domain.StockMovement
	levels    domain.StockLevelPage
	kardex    domain.MovementPage
	// products is the product of each variant the use cases return.
	products map[uuid.UUID]uuid.UUID
	err      error
}

func (s *stubUseCases) CreateLocation(_ context.Context, a app.Actor, name string) (*domain.Location, error) {
	s.called, s.actor, s.name = true, a, name
	return s.location, s.err
}

func (s *stubUseCases) ListLocations(_ context.Context, a app.Actor) ([]*domain.Location, error) {
	s.called, s.actor = true, a
	return s.locations, s.err
}

func (s *stubUseCases) ReceiveStock(_ context.Context, a app.Actor, in app.ReceiptInput) (app.Movement, error) {
	s.called, s.actor, s.receipt = true, a, in
	return s.withProduct(s.movement), s.err
}

func (s *stubUseCases) AdjustStock(_ context.Context, a app.Actor, in app.AdjustmentInput) (app.Movement, error) {
	s.called, s.actor, s.adjustment = true, a, in
	return s.withProduct(s.movement), s.err
}

func (s *stubUseCases) TransferStock(_ context.Context, a app.Actor, in app.TransferInput) (app.Movement, app.Movement, error) {
	s.called, s.actor, s.transfer = true, a, in
	return s.withProduct(s.out), s.withProduct(s.in), s.err
}

func (s *stubUseCases) ListStock(_ context.Context, a app.Actor, q app.StockQuery) (app.LevelPage, error) {
	s.called, s.actor, s.stockQuery = true, a, q
	page := app.LevelPage{Total: s.levels.Total}
	for _, l := range s.levels.Items {
		page.Items = append(page.Items, app.Level{StockLevel: l, ProductID: s.products[l.VariantID()]})
	}
	return page, s.err
}

func (s *stubUseCases) GetKardex(_ context.Context, a app.Actor, q app.KardexQuery) (app.MovementPage, error) {
	s.called, s.actor, s.kardexQuery = true, a, q
	page := app.MovementPage{Total: s.kardex.Total}
	for _, m := range s.kardex.Items {
		page.Items = append(page.Items, s.withProduct(m))
	}
	return page, s.err
}

// withProduct is m with the product products gives its variant, as the use
// cases return it.
func (s *stubUseCases) withProduct(m *domain.StockMovement) app.Movement {
	if m == nil {
		return app.Movement{}
	}
	return app.Movement{StockMovement: m, ProductID: s.products[m.VariantID()]}
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
		TenantID: owner.TenantID, VariantID: newID(), LocationID: newID(), Quantity: dec("10"), AverageCost: dec("1000"),
	})
	require.NoError(t, err)
	m, err := level.Receive(dec("2.5"), dec("1200.5"), domain.MovementInfo{UserID: owner.UserID, OccurredAt: now})
	require.NoError(t, err)
	return m
}
