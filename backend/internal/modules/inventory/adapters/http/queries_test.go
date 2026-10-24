package httpadapter_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestLocations(t *testing.T) {
	t.Run("crea una sede", func(t *testing.T) {
		loc, err := domain.NewLocation(owner.TenantID, "Bodega principal", now)
		require.NoError(t, err)
		uc := &stubUseCases{location: loc}

		rec := serve(t, uc, &owner, http.MethodPost, "/api/v1/locations", `{"name":"Bodega principal"}`)

		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		require.Equal(t, "Bodega principal", uc.name)
		got := decodeMap(t, rec)
		require.Equal(t, loc.ID().String(), got["id"])
		require.Equal(t, true, got["active"])
	})

	t.Run("nombre repetido responde 409 location_name_taken", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrLocationNameTaken}, &owner, http.MethodPost, "/api/v1/locations", `{"name":"Bodega"}`)
		requireError(t, rec, http.StatusConflict, "location_name_taken")
	})

	t.Run("lista las sedes", func(t *testing.T) {
		loc, err := domain.NewLocation(owner.TenantID, "Bodega", now)
		require.NoError(t, err)
		rec := serve(t, &stubUseCases{locations: []*domain.Location{loc}}, &owner, http.MethodGet, "/api/v1/locations", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Len(t, decodeMap(t, rec)["items"], 1)
	})
}

func TestListStock(t *testing.T) {
	locationID := newID()

	t.Run("lista las existencias con la paginación por defecto y oculta el costo a quien no edita el catálogo", func(t *testing.T) {
		level, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
			TenantID: owner.TenantID, VariantID: newID(), LocationID: locationID, Quantity: dec("7.5"), AverageCost: dec("100"),
		})
		require.NoError(t, err)
		uc := &stubUseCases{levels: domain.StockLevelPage{Items: []*domain.StockLevel{level}, Total: 1}}
		p := warehouse()

		rec := serve(t, uc, &p, http.MethodGet, "/api/v1/locations/"+locationID.String()+"/stock", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, app.StockQuery{LocationID: locationID, Limit: 20}, uc.stockQuery)
		require.Contains(t, rec.Body.String(), `"quantity":"7.5"`)
		require.Contains(t, rec.Body.String(), `"variant_id":"`+level.VariantID().String()+`"`)
		require.NotContains(t, rec.Body.String(), "average_cost")
	})

	t.Run("el dueño ve el costo promedio", func(t *testing.T) {
		level, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
			TenantID: owner.TenantID, VariantID: newID(), LocationID: locationID, Quantity: dec("1"), AverageCost: dec("100"),
		})
		require.NoError(t, err)
		uc := &stubUseCases{levels: domain.StockLevelPage{Items: []*domain.StockLevel{level}, Total: 1}}

		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/locations/"+locationID.String()+"/stock", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"average_cost":"100.00"`)
	})

	t.Run("pasa variante y paginación", func(t *testing.T) {
		variantID := newID()
		uc := &stubUseCases{}
		rec := serve(t, uc, &owner, http.MethodGet,
			"/api/v1/locations/"+locationID.String()+"/stock?variant_id="+variantID.String()+"&limit=5&offset=10", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, app.StockQuery{LocationID: locationID, VariantID: &variantID, Limit: 5, Offset: 10}, uc.stockQuery)
	})

	t.Run("sede inexistente responde 404 location_not_found", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrLocationNotFound}, &owner, http.MethodGet, "/api/v1/locations/"+locationID.String()+"/stock", "")
		requireError(t, rec, http.StatusNotFound, "location_not_found")
	})
}

func TestGetKardex(t *testing.T) {
	t.Run("el kardex de una variante pasa variante, sede y paginación y devuelve la página", func(t *testing.T) {
		variantID, locationID := newID(), newID()
		m := receipt(t)
		uc := &stubUseCases{kardex: domain.MovementPage{Items: []*domain.StockMovement{m}, Total: 3}}

		rec := serve(t, uc, &owner, http.MethodGet,
			"/api/v1/variants/"+variantID.String()+"/kardex?location_id="+locationID.String()+"&offset=2", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, app.KardexQuery{VariantID: variantID, LocationID: &locationID, Limit: 20, Offset: 2}, uc.kardexQuery)
		got := decodeMap(t, rec)
		require.EqualValues(t, 3, got["total"])
		require.Len(t, got["items"], 1)
		require.Equal(t, m.VariantID().String(), got["items"].([]any)[0].(map[string]any)["variant_id"])
	})

	t.Run("el kardex por producto ya no existe", func(t *testing.T) {
		rec := serve(t, &stubUseCases{}, &owner, http.MethodGet, "/api/v1/products/"+newID().String()+"/kardex", "")

		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	})
}
