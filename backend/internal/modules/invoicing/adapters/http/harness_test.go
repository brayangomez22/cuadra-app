package httpadapter_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

// stub implements the use cases the tests call; any other panics.
type stub struct {
	httpadapter.UseCases
	account    func(app.Actor) (domain.ProviderAccount, error)
	setAccount func(app.Actor, app.ProviderAccountInput) (domain.ProviderAccount, error)
	file       func(app.Actor, uuid.UUID, string) (domain.File, error)
}

func (s stub) ProviderAccount(_ context.Context, a app.Actor) (domain.ProviderAccount, error) {
	return s.account(a)
}

func (s stub) UpdateProviderAccount(_ context.Context, a app.Actor, in app.ProviderAccountInput) (domain.ProviderAccount, error) {
	return s.setAccount(a, in)
}

func (s stub) DocumentFile(_ context.Context, a app.Actor, id uuid.UUID, kind string) (domain.File, error) {
	return s.file(a, id, kind)
}

// owner is the authenticated caller of the requests.
var owner = auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "owner"}

// serve runs a request with body as the owner and returns the recorder.
func serve(t *testing.T, uc httpadapter.UseCases, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithPrincipal(req.Context(), owner))
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, slog.New(slog.DiscardHandler))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// requireError checks the status and the error code of a response.
func requireError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, rec.Code, rec.Body.String())
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, code, body.Error.Code)
}
