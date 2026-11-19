package httpadapter_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func TestListProductsSort(t *testing.T) {
	t.Run("pasa el orden por precio al caso de uso", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, cashier(), http.MethodGet, "/api/v1/products?sort=price", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, domain.SortByPrice, uc.filter.Sort)
	})
	t.Run("quien ve costos ordena por margen", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, owner, http.MethodGet, "/api/v1/products?sort=margin", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, domain.SortByMargin, uc.filter.Sort)
	})
	t.Run("ordenar por margen sin ver costos responde 403 costs_hidden", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, cashier(), http.MethodGet, "/api/v1/products?sort=margin", "")
		requireError(t, rec, http.StatusForbidden, "costs_hidden")
	})
}
