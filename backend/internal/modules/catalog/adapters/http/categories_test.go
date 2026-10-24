package httpadapter_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	httpadapter "github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/http"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestCategories(t *testing.T) {
	t.Run("crea una categoría con padre", func(t *testing.T) {
		parentID := uuid.Must(uuid.NewV7())
		c, err := domain.NewCategory(owner.TenantID, "Tubería", &parentID, now)
		require.NoError(t, err)
		uc := &stubUseCases{category: c}

		rec := serve(t, uc, owner, http.MethodPost, "/api/v1/categories", `{"name":"Tubería","parent_id":"`+parentID.String()+`"}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		body := decodeMap(t, rec)
		require.Equal(t, "Tubería", body["name"])
		require.Equal(t, parentID.String(), body["parent_id"])
		require.Equal(t, "Tubería", uc.name)
		require.Equal(t, &parentID, uc.parentID)
	})

	t.Run("lista las categorías", func(t *testing.T) {
		c, err := domain.NewCategory(owner.TenantID, "Plomería", nil, now)
		require.NoError(t, err)
		uc := &stubUseCases{list: []*domain.Category{c}}

		rec := serve(t, uc, cashier(), http.MethodGet, "/api/v1/categories", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"name":"Plomería"`)
		require.NotContains(t, rec.Body.String(), "parent_id")
	})

	t.Run("categoría con ciclo responde 422 category_cycle", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrCategoryCycle}
		rec := serve(t, uc, owner, http.MethodPut, "/api/v1/categories/"+uuid.NewString(), `{"name":"A","parent_id":"`+uuid.NewString()+`"}`)
		requireError(t, rec, http.StatusUnprocessableEntity, "category_cycle")
	})

	t.Run("borrar categoría en uso responde 409 category_in_use", func(t *testing.T) {
		uc := &stubUseCases{err: domain.ErrCategoryInUse}
		requireError(t, serve(t, uc, owner, http.MethodDelete, "/api/v1/categories/"+uuid.NewString(), ""), http.StatusConflict, "category_in_use")
	})

	t.Run("borrar categoría vacía responde 204", func(t *testing.T) {
		id := uuid.Must(uuid.NewV7())
		uc := &stubUseCases{}

		rec := serve(t, uc, owner, http.MethodDelete, "/api/v1/categories/"+id.String(), "")

		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.Equal(t, id, uc.id)
	})
}

func TestMissingPrincipal(t *testing.T) {
	t.Run("sin principal responde 401 y no llama al caso de uso", func(t *testing.T) {
		mux := http.NewServeMux()
		uc := &stubUseCases{}
		httpadapter.Register(mux, uc, canWrite, slog.New(slog.DiscardHandler))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil))

		requireError(t, rec, http.StatusUnauthorized, "unauthorized")
		require.Equal(t, app.Actor{}, uc.actor)
	})
}
