package httpadapter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

var owner = auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "owner"}

// stubUseCases records what the handlers pass and answers with its fields.
type stubUseCases struct {
	upload   app.UploadInput
	actor    app.Actor
	format   domain.Format
	preview  app.Preview
	imp      *domain.Import
	page     app.ErrorPage
	template []byte
	err      error
}

func (s *stubUseCases) UploadImport(_ context.Context, a app.Actor, in app.UploadInput) (app.Preview, error) {
	s.actor, s.upload = a, in
	return s.preview, s.err
}

func (s *stubUseCases) ConfirmImport(_ context.Context, a app.Actor, _ uuid.UUID) (*domain.Import, error) {
	s.actor = a
	return s.imp, s.err
}

func (s *stubUseCases) GetImport(_ context.Context, a app.Actor, _ uuid.UUID) (*domain.Import, error) {
	s.actor = a
	return s.imp, s.err
}

func (s *stubUseCases) ListImportErrors(_ context.Context, a app.Actor, _ uuid.UUID, _, _ int) (app.ErrorPage, error) {
	s.actor = a
	return s.page, s.err
}

func (s *stubUseCases) ImportTemplate(_ context.Context, a app.Actor, format domain.Format) ([]byte, error) {
	s.actor, s.format = a, format
	return s.template, s.err
}

func anImport(t *testing.T) *domain.Import {
	t.Helper()
	imp, err := domain.NewImport(domain.NewImportParams{
		TenantID: owner.TenantID, UserID: owner.UserID, LocationID: uuid.Must(uuid.NewV7()),
		FileName: "productos.xlsx", Format: domain.FormatXLSX, Policy: domain.UpdateExisting, Rows: 2, Products: 1,
	}, time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	return imp
}

func serve(t *testing.T, uc *stubUseCases, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, slog.New(slog.DiscardHandler))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req.WithContext(auth.WithPrincipal(req.Context(), owner)))
	return rec
}

// uploadRequest is a multipart upload of a file named name.
func uploadRequest(t *testing.T, name string, data []byte, fields map[string]string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	part, err := form.CreateFormFile("file", name)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	for k, v := range fields {
		require.NoError(t, form.WriteField(k, v))
	}
	require.NoError(t, form.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/imports", &buf)
	req.Header.Set("Content-Type", form.FormDataContentType())
	return req
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
