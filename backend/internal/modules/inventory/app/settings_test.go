package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestStockSettings(t *testing.T) {
	t.Run("una empresa sin reglas no deja el stock en negativo", func(t *testing.T) {
		e := newEnv(t)

		got, err := e.svc.StockSettings(t.Context(), e.actor)

		require.NoError(t, err)
		require.False(t, got.AllowNegative)
		_, attrs := e.span(t, "inventory.GetStockSettings")
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
	})
}

func TestUpdateStockSettings(t *testing.T) {
	yes, no := true, false

	t.Run("activa y desactiva el stock negativo en negocios distintos", func(t *testing.T) {
		// A store that sells before receiving, and one that never does.
		for _, allow := range []*bool{&yes, &no} {
			e := newEnv(t)

			got, err := e.svc.UpdateStockSettings(t.Context(), e.actor, app.StockSettingsUpdate{AllowNegative: allow})

			require.NoError(t, err)
			require.Equal(t, *allow, got.AllowNegative)
			require.Equal(t, domain.StockPolicy{AllowNegative: *allow}, e.store.policies[e.actor.TenantID])
			_, attrs := e.span(t, "inventory.UpdateStockSettings")
			require.Equal(t, e.actor.UserID.String(), attrs["user.id"].AsString())
			require.Equal(t, int64(1), e.counter(t, "cuadra.inventory.settings_changes", "setting", "allow_negative_stock"))
		}
	})

	t.Run("sin cambios devuelve las reglas como están y no guarda nada", func(t *testing.T) {
		e := newEnv(t)
		e.store.policies[e.actor.TenantID] = domain.StockPolicy{AllowNegative: true}

		got, err := e.svc.UpdateStockSettings(t.Context(), e.actor, app.StockSettingsUpdate{})

		require.NoError(t, err)
		require.True(t, got.AllowNegative)
		require.Zero(t, e.counter(t, "cuadra.inventory.settings_changes", "", ""))
	})

	t.Run("cambiar las reglas de una empresa no cambia las de otra", func(t *testing.T) {
		e := newEnv(t)
		other := app.Actor{TenantID: uuid.Must(uuid.NewV7()), UserID: uuid.Must(uuid.NewV7())}

		_, err := e.svc.UpdateStockSettings(t.Context(), e.actor, app.StockSettingsUpdate{AllowNegative: &yes})
		require.NoError(t, err)

		got, err := e.svc.StockSettings(t.Context(), other)
		require.NoError(t, err)
		require.False(t, got.AllowNegative)
	})
}
