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

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

var now = time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// stubUseCases records the last call and answers with what each test sets.
type stubUseCases struct {
	actor   app.Actor
	confirm app.ConfirmSaleInput
	id      uuid.UUID
	reason  string
	list    app.ListSalesInput

	openCash app.OpenCashSessionInput
	movement app.CashMovementRequest
	closing  app.CloseCashSessionInput

	sale     *domain.Sale
	page     domain.SalePage
	limit    domain.DiscountLimit
	session  *domain.CashSession
	view     app.CashSessionView
	current  app.CurrentCashSession
	recorded domain.CashMovement
	err      error
}

func (s *stubUseCases) GetDiscountLimit(_ context.Context, a app.Actor) (domain.DiscountLimit, error) {
	s.actor = a
	return s.limit, s.err
}

func (s *stubUseCases) ConfirmSale(_ context.Context, a app.Actor, in app.ConfirmSaleInput) (*domain.Sale, error) {
	s.actor, s.confirm = a, in
	return s.sale, s.err
}

func (s *stubUseCases) VoidSale(_ context.Context, a app.Actor, id uuid.UUID, reason string) (*domain.Sale, error) {
	s.actor, s.id, s.reason = a, id, reason
	return s.sale, s.err
}

func (s *stubUseCases) GetSale(_ context.Context, a app.Actor, id uuid.UUID) (*domain.Sale, error) {
	s.actor, s.id = a, id
	return s.sale, s.err
}

func (s *stubUseCases) ListSales(_ context.Context, a app.Actor, in app.ListSalesInput) (domain.SalePage, error) {
	s.actor, s.list = a, in
	return s.page, s.err
}

// discountOnlyOwner grants sales:discount only to owners.
func discountOnlyOwner(role, permission string) bool {
	return role == "owner" || permission != "sales:discount"
}

var owner = auth.Principal{TenantID: newID(), UserID: newID(), Role: "owner"}

func cashier() auth.Principal {
	p := owner
	p.Role = "cashier"
	return p
}

func serve(t *testing.T, uc *stubUseCases, p *auth.Principal, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, discountOnlyOwner, slog.New(slog.DiscardHandler))
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

// aSale is a stored sale: 2.5 × 8403.3613 at 19% with a discount of 100 of
// a variant with a label and a price list's price, plus 1 × 4500 at 0% of a
// simple product, paid in cash with 30000.
func aSale(t *testing.T) *domain.Sale {
	t.Helper()
	s, err := domain.NewSale(domain.NewSaleParams{ID: newID(), TenantID: owner.TenantID, LocationID: newID(), UserID: owner.UserID, CreatedAt: now})
	require.NoError(t, err)
	require.NoError(t, s.AddLine(domain.LineInput{
		ProductID: newID(), VariantID: newID(), ProductName: "Tubo PVC", VariantLabel: "1/2", Quantity: dec("2.5"),
		UnitPrice: dec("8403.3613"), TaxRate: dec("0.19"), Discount: dec("100"), PriceRule: "price_lists",
	}, domain.NoDiscountLimit))
	require.NoError(t, s.AddLine(domain.LineInput{
		ProductID: newID(), VariantID: newID(), ProductName: "Puntilla 2\"", Quantity: dec("1"), UnitPrice: dec("4500"),
		TaxRate: dec("0"), Discount: dec("0"), PriceRule: domain.VariantPriceRule,
	}, domain.NoDiscountLimit))
	cash, err := domain.NewPayment(domain.PaymentCash, dec("30000"))
	require.NoError(t, err)
	require.NoError(t, s.Confirm([]domain.Payment{cash}, now))
	require.NoError(t, s.AssignNumber(15))
	return s
}

type errorBody struct {
	Error struct {
		Code, Message string
		Details       map[string]string
	}
}

func (s *stubUseCases) OpenCashSession(_ context.Context, a app.Actor, in app.OpenCashSessionInput) (*domain.CashSession, error) {
	s.actor, s.openCash = a, in
	return s.session, s.err
}

func (s *stubUseCases) RecordCashMovement(_ context.Context, a app.Actor, id uuid.UUID, in app.CashMovementRequest) (domain.CashMovement, error) {
	s.actor, s.id, s.movement = a, id, in
	return s.recorded, s.err
}

func (s *stubUseCases) CloseCashSession(_ context.Context, a app.Actor, id uuid.UUID, in app.CloseCashSessionInput) (app.CashSessionView, error) {
	s.actor, s.id, s.closing = a, id, in
	return s.view, s.err
}

func (s *stubUseCases) GetCashSession(_ context.Context, a app.Actor, id uuid.UUID) (app.CashSessionView, error) {
	s.actor, s.id = a, id
	return s.view, s.err
}

func (s *stubUseCases) GetCurrentCashSession(_ context.Context, a app.Actor) (app.CurrentCashSession, error) {
	s.actor = a
	return s.current, s.err
}

// anOpenSession is the owner's session with a float of 100000 and a cash out
// of 12000.
func anOpenSession(t *testing.T) app.CashSessionView {
	t.Helper()
	s, err := domain.OpenCashSession(domain.NewCashSessionParams{
		ID: newID(), TenantID: owner.TenantID, LocationID: newID(), UserID: owner.UserID, OpeningFloat: dec("100000"), OpenedAt: now,
	})
	require.NoError(t, err)
	m, err := s.RecordMovement(domain.CashMovementInput{
		ID: newID(), Kind: domain.CashOut, Amount: dec("12000"), Reason: "Almuerzo", UserID: owner.UserID, At: now,
	})
	require.NoError(t, err)
	return app.CashSessionView{Session: s, Movements: []domain.CashMovement{m}}
}

// aClosedSession is anOpenSession closed with 150000 counted: expected
// 100000 + 80000 − 12000 − 20000 = 148000, overage 2000.
func aClosedSession(t *testing.T) app.CashSessionView {
	t.Helper()
	v := anOpenSession(t)
	count, err := domain.NewCashCount([]domain.DenominationCount{
		{Denomination: dec("100000"), Quantity: 1}, {Denomination: dec("50000"), Quantity: 1},
	})
	require.NoError(t, err)
	require.NoError(t, v.Session.Close(domain.CloseCashInput{
		UserID: owner.UserID, Count: count, Notes: "Sin novedad", At: now,
		Activity: domain.CashActivity{
			CashReceived: dec("80000"), ChangeGiven: dec("20000"), CashIn: dec("0"), CashOut: dec("12000"), Sales: 3,
			ByMethod: []domain.MethodTotal{{Method: domain.PaymentCash, Amount: dec("60000")}, {Method: domain.PaymentCard, Amount: dec("45000")}},
		},
	}))
	return v
}

type sessionBody struct {
	ID           uuid.UUID `json:"id"`
	Status       string    `json:"status"`
	OpeningFloat string    `json:"opening_float"`
	Movements    []struct {
		Kind, Amount, Reason string
	} `json:"movements"`
	Closing *struct {
		SalesCount   int    `json:"sales_count"`
		CashReceived string `json:"cash_received"`
		ChangeGiven  string `json:"change_given"`
		CashOut      string `json:"cash_out"`
		Expected     string `json:"expected"`
		Counted      string `json:"counted"`
		Difference   string `json:"difference"`
		Result       string `json:"result"`
		Notes        string `json:"notes"`
		Counts       []struct {
			Denomination string `json:"denomination"`
			Quantity     int64  `json:"quantity"`
		} `json:"counts"`
		Payments []struct{ Method, Amount string } `json:"payments"`
	} `json:"closing"`
}

func decodeSession(t *testing.T, raw []byte) sessionBody {
	t.Helper()
	var body sessionBody
	require.NoError(t, json.Unmarshal(raw, &body))
	return body
}
