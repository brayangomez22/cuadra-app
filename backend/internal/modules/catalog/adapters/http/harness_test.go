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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

var now = time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)

// stubUseCases records the last call and answers with what each test sets.
type stubUseCases struct {
	actor     app.Actor
	input     app.ProductInput
	variant   app.VariantUpdate
	filter    domain.ProductFilter
	id        uuid.UUID
	variantID uuid.UUID
	unitID    uuid.UUID
	unitInput app.VariantUnitInput
	name      string
	parentID  *uuid.UUID

	product  *domain.Product
	page     domain.ProductPage
	category *domain.Category
	list     []*domain.Category
	unit     *domain.UnitOfMeasure
	units    []*domain.UnitOfMeasure
	code     string
	err      error

	attrInput  app.AttributeInput
	attrUpdate app.AttributeUpdate
	attribute  *domain.AttributeDefinition
	attributes []*domain.AttributeDefinition
	axis       *domain.VariantAxis
	axes       []*domain.VariantAxis
	values     []string
}

func (s *stubUseCases) CreateProduct(_ context.Context, a app.Actor, in app.ProductInput) (*domain.Product, error) {
	s.actor, s.input = a, in
	return s.product, s.err
}

func (s *stubUseCases) UpdateProduct(_ context.Context, a app.Actor, id uuid.UUID, in app.ProductInput) (*domain.Product, error) {
	s.actor, s.id, s.input = a, id, in
	return s.product, s.err
}

func (s *stubUseCases) GetProduct(_ context.Context, a app.Actor, id uuid.UUID) (*domain.Product, error) {
	s.actor, s.id = a, id
	return s.product, s.err
}

func (s *stubUseCases) SearchProducts(_ context.Context, a app.Actor, f domain.ProductFilter) (domain.ProductPage, error) {
	s.actor, s.filter = a, f
	return s.page, s.err
}

func (s *stubUseCases) DeactivateProduct(_ context.Context, a app.Actor, id uuid.UUID) (*domain.Product, error) {
	s.actor, s.id = a, id
	return s.product, s.err
}

func (s *stubUseCases) ActivateProduct(_ context.Context, a app.Actor, id uuid.UUID) (*domain.Product, error) {
	s.actor, s.id = a, id
	return s.product, s.err
}

func (s *stubUseCases) AddVariant(_ context.Context, a app.Actor, productID uuid.UUID, in app.VariantInput) (*domain.Product, error) {
	s.actor, s.id, s.variant = a, productID, app.VariantUpdate{VariantInput: in}
	return s.product, s.err
}

func (s *stubUseCases) UpdateVariant(_ context.Context, a app.Actor, productID, variantID uuid.UUID, in app.VariantUpdate) (*domain.Product, error) {
	s.actor, s.id, s.variantID, s.variant = a, productID, variantID, in
	return s.product, s.err
}

func (s *stubUseCases) SetVariantUnit(_ context.Context, a app.Actor, productID, variantID, unitID uuid.UUID, in app.VariantUnitInput) (*domain.Product, error) {
	s.actor, s.id, s.variantID, s.unitID, s.unitInput = a, productID, variantID, unitID, in
	return s.product, s.err
}

func (s *stubUseCases) RemoveVariantUnit(_ context.Context, a app.Actor, productID, variantID, unitID uuid.UUID) (*domain.Product, error) {
	s.actor, s.id, s.variantID, s.unitID = a, productID, variantID, unitID
	return s.product, s.err
}

func (s *stubUseCases) CreateCategory(_ context.Context, a app.Actor, name string, parentID *uuid.UUID) (*domain.Category, error) {
	s.actor, s.name, s.parentID = a, name, parentID
	return s.category, s.err
}

func (s *stubUseCases) UpdateCategory(_ context.Context, a app.Actor, id uuid.UUID, name string, parentID *uuid.UUID) (*domain.Category, error) {
	s.actor, s.id, s.name, s.parentID = a, id, name, parentID
	return s.category, s.err
}

func (s *stubUseCases) ListCategories(_ context.Context, a app.Actor) ([]*domain.Category, error) {
	s.actor = a
	return s.list, s.err
}

func (s *stubUseCases) DeleteCategory(_ context.Context, a app.Actor, id uuid.UUID) error {
	s.actor, s.id = a, id
	return s.err
}

func (s *stubUseCases) CreateUnit(_ context.Context, a app.Actor, code, name string) (*domain.UnitOfMeasure, error) {
	s.actor, s.code, s.name = a, code, name
	return s.unit, s.err
}

func (s *stubUseCases) UpdateUnit(_ context.Context, a app.Actor, id uuid.UUID, code, name string) (*domain.UnitOfMeasure, error) {
	s.actor, s.id, s.code, s.name = a, id, code, name
	return s.unit, s.err
}

func (s *stubUseCases) ListUnits(_ context.Context, a app.Actor) ([]*domain.UnitOfMeasure, error) {
	s.actor = a
	return s.units, s.err
}

func (s *stubUseCases) DeleteUnit(_ context.Context, a app.Actor, id uuid.UUID) error {
	s.actor, s.id = a, id
	return s.err
}

// canWrite grants catalog:write only to owners, like a reduced permission matrix.
func canWrite(role, permission string) bool { return role == "owner" || permission == "catalog:read" }

var owner = auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "owner"}

func cashier() auth.Principal {
	p := owner
	p.Role = "cashier"
	return p
}

func serve(t *testing.T, uc *stubUseCases, p auth.Principal, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec, _ := serveLabeled(t, uc, p, method, path, body)
	return rec
}

// serveLabeled also returns the otelhttp labeler of the request, which
// collects the attributes the handler adds to the HTTP metrics.
func serveLabeled(t *testing.T, uc *stubUseCases, p auth.Principal, method, path, body string) (*httptest.ResponseRecorder, *otelhttp.Labeler) {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, canWrite, slog.New(slog.DiscardHandler))
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	labeler := &otelhttp.Labeler{}
	req = req.WithContext(otelhttp.ContextWithLabeler(auth.WithPrincipal(req.Context(), p), labeler))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec, labeler
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

// meters is the owner's tenant unit the test products use.
var meters = mustUnit("m", "Metro")

func mustUnit(code, name string) *domain.UnitOfMeasure {
	u, err := domain.NewUnit(owner.TenantID, code, name, now)
	if err != nil {
		panic(err)
	}
	return u
}

func product(t *testing.T, cost, price string) *domain.Product {
	t.Helper()
	rate, err := domain.ParseTaxRate(decimal.RequireFromString("0.19"))
	require.NoError(t, err)
	p, err := domain.NewProduct(domain.NewProductParams{
		TenantID: owner.TenantID, Name: "Tubo PVC presión 1/2 pulgada", BaseUnit: meters.Ref(), TaxRate: rate,
		Variant: domain.VariantParams{
			SKU: "TUB-PVC-12", Barcode: "7701234567890",
			Cost: decimal.RequireFromString(cost), Price: decimal.RequireFromString(price),
		},
	}, now)
	require.NoError(t, err)
	return p
}

var productBody = `{"sku":"tub-pvc-12","barcode":"7701234567890","name":"Tubo PVC presión 1/2 pulgada","base_unit_id":"` + meters.ID().String() + `","cost":"8333.33","price":"10000","tax_rate":"0.19","attributes":{}}`
