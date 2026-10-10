package domain_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestNewLocation(t *testing.T) {
	t.Run("crea sede activa con id v7 y nombre recortado", func(t *testing.T) {
		tenantID := newID()
		loc, err := domain.NewLocation(tenantID, "  Bodega principal  ", testNow)
		require.NoError(t, err)
		require.Equal(t, 7, int(loc.ID().Version()))
		require.Equal(t, tenantID, loc.TenantID())
		require.Equal(t, "Bodega principal", loc.Name())
		require.True(t, loc.IsActive())
		require.Equal(t, testNow.UTC(), loc.CreatedAt())
	})

	t.Run("rechaza nombre vacío o demasiado largo", func(t *testing.T) {
		for _, name := range []string{"", "   ", strings.Repeat("a", 101)} {
			_, err := domain.NewLocation(newID(), name, testNow)
			require.ErrorIs(t, err, domain.ErrInvalidLocationName)
		}
	})

	t.Run("rechaza tenant vacío", func(t *testing.T) {
		_, err := domain.NewLocation(uuid.Nil, "Sede centro", testNow)
		require.ErrorIs(t, err, domain.ErrInvalidTenantID)
	})
}

func TestLocation(t *testing.T) {
	t.Run("renombra y conserva el nombre si el nuevo es inválido", func(t *testing.T) {
		loc, err := domain.NewLocation(newID(), "Sede centro", testNow)
		require.NoError(t, err)

		require.NoError(t, loc.Rename(" Sede norte "))
		require.Equal(t, "Sede norte", loc.Name())
		require.ErrorIs(t, loc.Rename(" "), domain.ErrInvalidLocationName)
		require.Equal(t, "Sede norte", loc.Name())
	})

	t.Run("se desactiva y se reactiva", func(t *testing.T) {
		loc, err := domain.NewLocation(newID(), "Sede centro", testNow)
		require.NoError(t, err)

		loc.Deactivate()
		require.False(t, loc.IsActive())
		loc.Activate()
		require.True(t, loc.IsActive())
	})

	t.Run("rehidrata una sede guardada y rechaza id vacío", func(t *testing.T) {
		s := domain.LocationSnapshot{ID: newID(), TenantID: newID(), Name: "Bodega", Active: false, CreatedAt: testNow}
		loc, err := domain.RehydrateLocation(s)
		require.NoError(t, err)
		require.Equal(t, s.ID, loc.ID())
		require.False(t, loc.IsActive())

		s.ID = uuid.Nil
		_, err = domain.RehydrateLocation(s)
		require.ErrorIs(t, err, domain.ErrInvalidLocationID)
	})
}

func TestLocationRequireActive(t *testing.T) {
	t.Run("una sede activa acepta movimientos", func(t *testing.T) {
		loc, err := domain.NewLocation(newID(), "Sede centro", testNow)
		require.NoError(t, err)
		require.NoError(t, loc.RequireActive())
	})

	t.Run("una sede inactiva rechaza movimientos con ErrLocationInactive", func(t *testing.T) {
		loc, err := domain.NewLocation(newID(), "Sede centro", testNow)
		require.NoError(t, err)
		loc.Deactivate()
		require.ErrorIs(t, loc.RequireActive(), domain.ErrLocationInactive)
	})
}
