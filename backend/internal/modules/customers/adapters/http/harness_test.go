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
	"github.com/stretchr/testify/require"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

var now = time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)

var owner = auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "owner"}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// stubUseCases records the last call and answers with what each test sets.
type stubUseCases struct {
	actor  app.Actor
	id     uuid.UUID
	params domain.CustomerParams
	query  app.CustomerQuery
	name   string

	customer *domain.Customer
	page     app.CustomerPage
	segment  *domain.Segment
	segments []*domain.Segment
	err      error
}

func (s *stubUseCases) CreateCustomer(_ context.Context, a app.Actor, p domain.CustomerParams) (*domain.Customer, error) {
	s.actor, s.params = a, p
	return s.customer, s.err
}

func (s *stubUseCases) UpdateCustomer(_ context.Context, a app.Actor, id uuid.UUID, p domain.CustomerParams) (*domain.Customer, error) {
	s.actor, s.id, s.params = a, id, p
	return s.customer, s.err
}

func (s *stubUseCases) GetCustomer(_ context.Context, a app.Actor, id uuid.UUID) (*domain.Customer, error) {
	s.actor, s.id = a, id
	return s.customer, s.err
}

func (s *stubUseCases) ListCustomers(_ context.Context, a app.Actor, q app.CustomerQuery) (app.CustomerPage, error) {
	s.actor, s.query = a, q
	return s.page, s.err
}

func (s *stubUseCases) CreateSegment(_ context.Context, a app.Actor, name string) (*domain.Segment, error) {
	s.actor, s.name = a, name
	return s.segment, s.err
}

func (s *stubUseCases) RenameSegment(_ context.Context, a app.Actor, id uuid.UUID, name string) (*domain.Segment, error) {
	s.actor, s.id, s.name = a, id, name
	return s.segment, s.err
}

func (s *stubUseCases) DeleteSegment(_ context.Context, a app.Actor, id uuid.UUID) error {
	s.actor, s.id = a, id
	return s.err
}

func (s *stubUseCases) ListSegments(_ context.Context, a app.Actor) ([]*domain.Segment, error) {
	s.actor = a
	return s.segments, s.err
}

// aCompany is a customer with every field set.
func aCompany(t *testing.T) *domain.Customer {
	t.Helper()
	segment := newID()
	c, err := domain.NewCustomer(owner.TenantID, domain.CustomerParams{
		PersonType: domain.PersonLegal, DocumentType: domain.DocumentNIT, DocumentNumber: "900123456-8",
		Name: "Confecciones La Moda S.A.S.", TradeName: "La Moda", SegmentID: &segment,
		Contact:             domain.Contact{Email: "compras@lamoda.co", Phone: "604 444 5566", Address: "Cra 50 # 10-20", City: "Medellín"},
		TaxResponsibilities: []string{"O-13"}, Active: true,
	}, now)
	require.NoError(t, err)
	return c
}

// aPerson is a customer with only the required fields.
func aPerson(t *testing.T) *domain.Customer {
	t.Helper()
	c, err := domain.NewCustomer(owner.TenantID, domain.CustomerParams{
		PersonType: domain.PersonNatural, DocumentType: domain.DocumentCC, DocumentNumber: "1020304050",
		Name: "Carlos Pérez", Active: true,
	}, now)
	require.NoError(t, err)
	return c
}

func serve(t *testing.T, uc *stubUseCases, p *auth.Principal, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, slog.New(slog.DiscardHandler))
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

// requireError checks the status and the error's code and returns its
// message.
func requireError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) string {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, code, body.Error.Code)
	return body.Error.Message
}
