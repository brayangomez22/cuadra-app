package httpadapter_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

var admin = auth.Principal{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7()), Role: "admin"}

type stubUseCases struct {
	actor     app.Actor
	checklist domain.Checklist
	err       error
}

func (s *stubUseCases) GetChecklist(_ context.Context, a app.Actor) (domain.Checklist, error) {
	s.actor = a
	return s.checklist, s.err
}

func (s *stubUseCases) DismissChecklist(_ context.Context, a app.Actor) error {
	s.actor = a
	return s.err
}

func do(t *testing.T, uc *stubUseCases, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	httpadapter.Register(mux, uc, slog.New(slog.DiscardHandler))
	req := httptest.NewRequest(method, path, nil)
	req = req.WithContext(auth.WithPrincipal(req.Context(), admin))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestGetOnboardingChecklist(t *testing.T) {
	t.Run("devuelve los pasos del usuario del token con su progreso", func(t *testing.T) {
		uc := &stubUseCases{checklist: domain.Checklist{Steps: []domain.StepState{
			{ID: domain.StepProducts, Done: true},
			{ID: domain.StepInvoicing, Critical: true},
		}}}

		rec := do(t, uc, http.MethodGet, "/api/v1/onboarding/checklist")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"visible": true, "dismissed": false, "completed": 1, "total": 2, "steps": [
			{"id": "products", "done": true, "critical": false},
			{"id": "invoicing", "done": false, "critical": true}
		]}`, rec.Body.String())
		require.Equal(t, app.Actor{TenantID: admin.TenantID, UserID: admin.UserID, Role: "admin"}, uc.actor)
	})
	t.Run("una lista sin pasos viene vacía y no visible", func(t *testing.T) {
		rec := do(t, &stubUseCases{}, http.MethodGet, "/api/v1/onboarding/checklist")

		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"visible": false, "dismissed": false, "completed": 0, "total": 0, "steps": []}`, rec.Body.String())
	})
}

func TestDismissOnboardingChecklist(t *testing.T) {
	t.Run("oculta la lista del usuario del token", func(t *testing.T) {
		uc := &stubUseCases{}

		rec := do(t, uc, http.MethodPut, "/api/v1/onboarding/checklist/dismissal")

		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.Equal(t, admin.UserID, uc.actor.UserID)
	})
	t.Run("sin pasos que ocultar responde 409", func(t *testing.T) {
		rec := do(t, &stubUseCases{err: domain.ErrNothingToDismiss}, http.MethodPut, "/api/v1/onboarding/checklist/dismissal")

		require.Equal(t, http.StatusConflict, rec.Code)
		require.Contains(t, rec.Body.String(), `"nothing_to_dismiss"`)
	})
}
