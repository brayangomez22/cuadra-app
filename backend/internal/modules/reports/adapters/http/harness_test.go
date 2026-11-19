package httpadapter_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/reports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

var (
	owner = auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "owner"}
	admin = auth.Principal{TenantID: owner.TenantID, UserID: uuid.Must(uuid.NewV7()), Role: "admin"}
)

// onlyOwnerSeesCosts is the permission matrix of the tests: the admin does
// not see costs.
func onlyOwnerSeesCosts(role, _ string) bool { return role == "owner" }

// stubUseCases records the last actor and query and answers with what each
// test sets.
type stubUseCases struct {
	actor    app.Actor
	sales    app.SalesQuery
	products app.ProductsQuery
	exported []string
	err      error

	salesReport    domain.SalesReport
	productsReport domain.ProductsReport
	aging          domain.AgingExport
}

func (s *stubUseCases) SalesReport(_ context.Context, a app.Actor, q app.SalesQuery) (domain.SalesReport, error) {
	s.actor, s.sales = a, q
	return s.salesReport, s.err
}

func (s *stubUseCases) ProductsReport(_ context.Context, a app.Actor, q app.ProductsQuery) (domain.ProductsReport, error) {
	s.actor, s.products = a, q
	return s.productsReport, s.err
}

func (s *stubUseCases) SaleProfits(_ context.Context, a app.Actor, _ app.SaleProfitsQuery) (domain.SaleProfitsReport, error) {
	s.actor = a
	return domain.SaleProfitsReport{}, s.err
}

func (s *stubUseCases) IdleStock(_ context.Context, a app.Actor, _ app.IdleQuery) (domain.IdleStockReport, error) {
	s.actor = a
	return domain.IdleStockReport{}, s.err
}

func (s *stubUseCases) StockAdjustments(_ context.Context, a app.Actor, _ app.AdjustmentsQuery) (domain.AdjustmentsReport, error) {
	s.actor = a
	return domain.AdjustmentsReport{}, s.err
}

func (s *stubUseCases) CashClosings(_ context.Context, a app.Actor, _ app.CashQuery) (domain.CashReport, error) {
	s.actor = a
	return domain.CashReport{}, s.err
}

func (s *stubUseCases) TeamControl(_ context.Context, a app.Actor, _ app.TeamQuery) (domain.TeamReport, error) {
	s.actor = a
	return domain.TeamReport{}, s.err
}

func (s *stubUseCases) ReceivablesAging(_ context.Context, a app.Actor, _ string) (domain.AgingExport, error) {
	s.actor = a
	return s.aging, s.err
}

func (s *stubUseCases) Exported(_ context.Context, report, format string) {
	s.exported = append(s.exported, report+"."+format)
}

func serve(t *testing.T, uc *stubUseCases, who auth.Principal, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, onlyOwnerSeesCosts, slog.New(slog.DiscardHandler))
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), who))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// requireError checks the status and the error's code.
func requireError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, code, body.Error.Code)
}
