package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func (e *env) createLocation(t *testing.T, name string) *domain.Location {
	t.Helper()
	l, err := e.svc.CreateLocation(t.Context(), e.actor, name)
	require.NoError(t, err)
	return l
}

func (e *env) receive(t *testing.T, variantID, locationID uuid.UUID, qty, cost string) *domain.StockMovement {
	t.Helper()
	m, err := e.svc.ReceiveStock(t.Context(), e.actor, app.ReceiptInput{
		VariantID: variantID, LocationID: locationID, Quantity: dec(qty), UnitCost: dec(cost),
	})
	require.NoError(t, err)
	return m.StockMovement
}

// deactivate takes a stored location out of use; there is no use case for it yet.
func (e *env) deactivate(id uuid.UUID) {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	e.store.locations[id].Deactivate()
}

func TestNewService(t *testing.T) {
	t.Run("falla si falta un repositorio", func(t *testing.T) {
		_, err := app.NewService(app.Deps{})
		require.Error(t, err)
	})
}
