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

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

var now = time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)

var owner = auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "owner"}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// stubUseCases records the last call and answers with what each test sets.
type stubUseCases struct {
	actor         app.Actor
	id            uuid.UUID
	params        domain.SupplierParams
	supplierQuery app.SupplierQuery
	input         app.PurchaseInput
	purchaseQuery app.PurchaseQuery

	supplier  app.SupplierView
	suppliers app.SupplierPage
	purchase  *domain.Purchase
	purchases app.PurchasePage
	err       error
}

func (s *stubUseCases) CreateSupplier(_ context.Context, a app.Actor, p domain.SupplierParams) (app.SupplierView, error) {
	s.actor, s.params = a, p
	return s.supplier, s.err
}

func (s *stubUseCases) UpdateSupplier(_ context.Context, a app.Actor, id uuid.UUID, p domain.SupplierParams) (app.SupplierView, error) {
	s.actor, s.id, s.params = a, id, p
	return s.supplier, s.err
}

func (s *stubUseCases) GetSupplier(_ context.Context, a app.Actor, id uuid.UUID) (app.SupplierView, error) {
	s.actor, s.id = a, id
	return s.supplier, s.err
}

func (s *stubUseCases) ListSuppliers(_ context.Context, a app.Actor, q app.SupplierQuery) (app.SupplierPage, error) {
	s.actor, s.supplierQuery = a, q
	return s.suppliers, s.err
}

func (s *stubUseCases) RegisterPurchase(_ context.Context, a app.Actor, in app.PurchaseInput) (*domain.Purchase, error) {
	s.actor, s.input = a, in
	return s.purchase, s.err
}

func (s *stubUseCases) GetPurchase(_ context.Context, a app.Actor, id uuid.UUID) (*domain.Purchase, error) {
	s.actor, s.id = a, id
	return s.purchase, s.err
}

func (s *stubUseCases) ListPurchases(_ context.Context, a app.Actor, q app.PurchaseQuery) (app.PurchasePage, error) {
	s.actor, s.purchaseQuery = a, q
	return s.purchases, s.err
}

// aSupplier is a supplier with every field set, owed balance.
func aSupplier(t *testing.T, balance string) app.SupplierView {
	t.Helper()
	s, err := domain.NewSupplier(owner.TenantID, domain.SupplierParams{
		Name: "Distribuidora del Valle", NIT: "900123456-8", PaymentTermDays: 30, Active: true,
		Contact: domain.Contact{Name: "Laura Pérez", Email: "ventas@valle.co", Phone: "604 444 5566"},
	}, now)
	require.NoError(t, err)
	return app.SupplierView{Supplier: s, BalanceDue: dec(balance)}
}

// aPurchase is a purchase on credit of 2 boxes of 100 at 45.000 plus VAT,
// received.
func aPurchase(t *testing.T) *domain.Purchase {
	t.Helper()
	box := newID()
	p, err := domain.NewPurchase(owner.TenantID, domain.PurchaseParams{
		Supplier: aSupplier(t, "0").Supplier, LocationID: newID(), InvoiceNumber: "FE 1234",
		IssuedOn: domain.NewDate(2026, time.October, 20), Payment: domain.PaymentCredit, CreatedBy: owner.UserID,
	}, []domain.LineInput{{
		VariantID: newID(), ProductName: "Tornillo drywall 6x1", Quantity: dec("2"), UnitCost: dec("45000"),
		TaxRate: dec("0.19"), Unit: domain.LineUnit{ID: &box, Code: "caja", Factor: dec("100")},
	}}, now)
	require.NoError(t, err)
	require.NoError(t, p.Receive([]domain.Received{{BaseQuantity: dec("200"), BaseUnitCost: dec("450")}}))
	return p
}

func serve(t *testing.T, uc *stubUseCases, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, slog.New(slog.DiscardHandler))
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req = req.WithContext(auth.WithPrincipal(req.Context(), owner))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// errorBody is the Error schema.
type errorBody struct {
	Error struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Details map[string]string `json:"details"`
	} `json:"error"`
}

func requireError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) errorBody {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	var body errorBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, code, body.Error.Code)
	return body
}
