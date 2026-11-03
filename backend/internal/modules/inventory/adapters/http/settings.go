package httpadapter

import (
	"context"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/adapters/http/inventoryapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// SettingsUseCases are the use cases of the tenant's stock rules.
type SettingsUseCases interface {
	StockSettings(ctx context.Context, a app.Actor) (domain.StockPolicy, error)
	UpdateStockSettings(ctx context.Context, a app.Actor, in app.StockSettingsUpdate) (domain.StockPolicy, error)
}

// settingsHandler serves the tenant's stock rules.
type settingsHandler struct{ uc SettingsUseCases }

// GetInventorySettings returns the tenant's stock rules.
func (h settingsHandler) GetInventorySettings(ctx context.Context, _ inventoryapi.GetInventorySettingsRequestObject) (inventoryapi.GetInventorySettingsResponseObject, error) {
	policy, err := h.uc.StockSettings(ctx, actor(ctx))
	if err != nil {
		return nil, err
	}
	return inventoryapi.GetInventorySettings200JSONResponse(stockSettings(policy)), nil
}

// UpdateInventorySettings changes the tenant's stock rules.
func (h settingsHandler) UpdateInventorySettings(ctx context.Context, req inventoryapi.UpdateInventorySettingsRequestObject) (inventoryapi.UpdateInventorySettingsResponseObject, error) {
	policy, err := h.uc.UpdateStockSettings(ctx, actor(ctx), app.StockSettingsUpdate{AllowNegative: req.Body.AllowNegativeStock})
	if err != nil {
		return nil, err
	}
	return inventoryapi.UpdateInventorySettings200JSONResponse(stockSettings(policy)), nil
}

func stockSettings(p domain.StockPolicy) inventoryapi.InventorySettings {
	return inventoryapi.InventorySettings{AllowNegativeStock: p.AllowNegative}
}
