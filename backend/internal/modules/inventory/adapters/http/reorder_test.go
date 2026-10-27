package httpadapter_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func reorderRule(t *testing.T, variantID, locationID uuid.UUID, minQty, maxQty string) *domain.ReorderRule {
	t.Helper()
	r, err := domain.NewReorderRule(domain.ReorderRuleSpec{
		TenantID: owner.TenantID, VariantID: variantID, LocationID: locationID, Min: dec(minQty), Max: dec(maxQty),
	})
	require.NoError(t, err)
	return r
}

func rulePath(locationID, variantID uuid.UUID) string {
	return "/api/v1/locations/" + locationID.String() + "/reorder-rules/" + variantID.String()
}

func TestSetReorderRule(t *testing.T) {
	locationID, variantID := newID(), newID()

	t.Run("guarda el mínimo y el máximo y responde la regla", func(t *testing.T) {
		uc := &stubUseCases{rule: reorderRule(t, variantID, locationID, "2.5", "30")}
		p := warehouse()

		rec := serve(t, uc, &p, http.MethodPut, rulePath(locationID, variantID), `{"min_quantity":"2.5","max_quantity":"30"}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, p.TenantID, uc.actor.TenantID)
		require.Equal(t, variantID, uc.reorderIn.VariantID)
		require.Equal(t, locationID, uc.reorderIn.LocationID)
		require.True(t, dec("2.5").Equal(uc.reorderIn.Min))
		require.True(t, dec("30").Equal(uc.reorderIn.Max))
		got := decodeMap(t, rec)
		require.Equal(t, "2.5", got["min_quantity"])
		require.Equal(t, "30", got["max_quantity"])
	})

	t.Run("rango inválido responde 422 invalid_reorder_range", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrInvalidReorderRange}, &owner, http.MethodPut,
			rulePath(locationID, variantID), `{"min_quantity":"8","max_quantity":"5"}`)
		requireError(t, rec, http.StatusUnprocessableEntity, "invalid_reorder_range")
	})

	t.Run("sede o variante inexistentes responden 404", func(t *testing.T) {
		for err, code := range map[error]string{
			domain.ErrLocationNotFound: "location_not_found",
			domain.ErrVariantNotFound:  "variant_not_found",
		} {
			rec := serve(t, &stubUseCases{err: err}, &owner, http.MethodPut,
				rulePath(locationID, variantID), `{"min_quantity":"1","max_quantity":"2"}`)
			requireError(t, rec, http.StatusNotFound, code)
		}
	})

	t.Run("una cantidad que no se puede leer responde 422 sin llamar al caso de uso", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, &owner, http.MethodPut, rulePath(locationID, variantID), `{"min_quantity":"uno","max_quantity":"2"}`)
		requireError(t, rec, http.StatusUnprocessableEntity, "invalid_reorder_range")
		require.False(t, uc.called)
	})
}

func TestDeleteReorderRule(t *testing.T) {
	locationID, variantID := newID(), newID()

	t.Run("quita la regla y responde 204", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, &owner, http.MethodDelete, rulePath(locationID, variantID), "")
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.Equal(t, [2]uuid.UUID{variantID, locationID}, uc.reorderKey)
	})

	t.Run("sin regla responde 404 reorder_rule_not_found", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrReorderRuleNotFound}, &owner, http.MethodDelete, rulePath(locationID, variantID), "")
		requireError(t, rec, http.StatusNotFound, "reorder_rule_not_found")
	})
}

func TestListReorderAlerts(t *testing.T) {
	t.Run("lista las alertas de una sede con lo que hay que pedir", func(t *testing.T) {
		locationID, variantID, productID := newID(), newID(), newID()
		alert, ok := reorderRule(t, variantID, locationID, "10", "50").Alert(dec("4"))
		require.True(t, ok)
		uc := &stubUseCases{alerts: app.AlertPage{Items: []app.Alert{{ReorderAlert: alert, ProductID: productID}}, Total: 1}}

		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/inventory/reorder-alerts?location_id="+locationID.String()+"&limit=5", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, app.ReorderQuery{LocationID: &locationID, Limit: 5}, uc.alertsQuery)
		got := decodeMap(t, rec)
		require.InDelta(t, 1, got["total"], 0)
		item := got["items"].([]any)[0].(map[string]any)
		require.Equal(t, productID.String(), item["product_id"])
		require.Equal(t, "4", item["quantity"])
		require.Equal(t, "10", item["min_quantity"])
		require.Equal(t, "50", item["max_quantity"])
		require.Equal(t, "46", item["suggested_quantity"])
	})

	t.Run("sin sede pide las alertas de todas", func(t *testing.T) {
		uc := &stubUseCases{}
		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/inventory/reorder-alerts", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, app.ReorderQuery{Limit: 20}, uc.alertsQuery)
	})

	t.Run("sede inexistente responde 422 location_not_found", func(t *testing.T) {
		rec := serve(t, &stubUseCases{err: domain.ErrLocationNotFound}, &owner, http.MethodGet,
			"/api/v1/inventory/reorder-alerts?location_id="+newID().String(), "")
		requireError(t, rec, http.StatusUnprocessableEntity, "location_not_found")
	})
}

func TestListStockReorder(t *testing.T) {
	t.Run("cada existencia trae su mínimo, su máximo y si hay que reponer", func(t *testing.T) {
		locationID, low, plenty, free := newID(), newID(), newID(), newID()
		var levels []*domain.StockLevel
		for variantID, qty := range map[uuid.UUID]string{low: "1", plenty: "40", free: "3"} {
			l, err := domain.RehydrateStockLevel(domain.StockLevelSnapshot{
				TenantID: owner.TenantID, VariantID: variantID, LocationID: locationID, Quantity: dec(qty),
			})
			require.NoError(t, err)
			levels = append(levels, l)
		}
		uc := &stubUseCases{
			levels: domain.StockLevelPage{Items: levels, Total: 3},
			rules: map[uuid.UUID]*domain.ReorderRule{
				low:    reorderRule(t, low, locationID, "2", "6"),
				plenty: reorderRule(t, plenty, locationID, "2", "6"),
			},
		}

		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/locations/"+locationID.String()+"/stock", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		byVariant := map[string]map[string]any{}
		for _, it := range decodeMap(t, rec)["items"].([]any) {
			item := it.(map[string]any)
			byVariant[item["variant_id"].(string)] = item
		}
		require.Equal(t, map[string]any{"min_quantity": "2", "max_quantity": "6", "needs_reorder": true}, byVariant[low.String()]["reorder"])
		require.Equal(t, false, byVariant[plenty.String()]["reorder"].(map[string]any)["needs_reorder"])
		require.NotContains(t, byVariant[free.String()], "reorder")
	})
}
