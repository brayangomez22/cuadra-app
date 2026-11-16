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

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/audit/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

var owner = auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "owner"}

// stubUseCases records the last call and answers with what each test sets.
type stubUseCases struct {
	actor app.Actor
	query app.EventQuery
	page  domain.EventPage
	err   error
}

func (s *stubUseCases) ListEvents(_ context.Context, a app.Actor, q app.EventQuery) (domain.EventPage, error) {
	s.actor, s.query = a, q
	return s.page, s.err
}

func serve(t *testing.T, uc *stubUseCases, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, slog.New(slog.DiscardHandler))
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), owner))
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
