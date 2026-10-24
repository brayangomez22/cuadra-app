package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestLocations(t *testing.T) {
	t.Run("crea una sede activa", func(t *testing.T) {
		e := newEnv(t)

		l, err := e.svc.CreateLocation(t.Context(), e.actor, " Bodega principal ")
		require.NoError(t, err)

		require.Equal(t, e.actor.TenantID, l.TenantID())
		require.Equal(t, "Bodega principal", l.Name())
		require.True(t, l.IsActive())
		require.Equal(t, testNow, l.CreatedAt())
		_, attrs := e.span(t, "inventory.CreateLocation")
		require.Equal(t, l.ID().String(), attrs["location.id"].AsString())
	})

	t.Run("rechaza sede con nombre repetido", func(t *testing.T) {
		e := newEnv(t)
		e.createLocation(t, "Bodega principal")

		_, err := e.svc.CreateLocation(t.Context(), e.actor, "bodega PRINCIPAL")
		require.ErrorIs(t, err, domain.ErrLocationNameTaken)
	})

	t.Run("rechaza nombre vacío", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.CreateLocation(t.Context(), e.actor, "  ")
		require.ErrorIs(t, err, domain.ErrInvalidLocationName)
	})

	t.Run("lista las sedes del tenant por nombre", func(t *testing.T) {
		e := newEnv(t)
		e.createLocation(t, "Sede norte")
		e.createLocation(t, "Bodega")

		list, err := e.svc.ListLocations(t.Context(), e.actor)
		require.NoError(t, err)
		require.Len(t, list, 2)
		require.Equal(t, "Bodega", list[0].Name())
		require.Equal(t, "Sede norte", list[1].Name())
	})
}
