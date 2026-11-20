package ports

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/modules"
)

func TestLoadTemplates(t *testing.T) {
	registry, err := modules.NewRegistry(modules.Builtin()...)
	require.NoError(t, err)
	catalogs, err := LoadTemplates(registry)
	require.NoError(t, err)
	steps := OnboardingTemplates{Catalog: catalogs}

	t.Run("cada plantilla embebida declara primeros pasos del catálogo", func(t *testing.T) {
		for _, tpl := range catalogs.List() {
			got, ok := steps.Steps(tpl.ID)
			require.True(t, ok, tpl.ID)
			require.NotEmpty(t, got, tpl.ID)
			require.Equal(t, "first_sale", got[len(got)-1], "%s termina con la primera venta", tpl.ID)
		}
	})
	t.Run("ferretería y ropa y calzado tienen pasos distintos", func(t *testing.T) {
		hardware, _ := steps.Steps("ferreteria")
		clothing, _ := steps.Steps("ropa_calzado")
		require.Contains(t, hardware, "alternate_units")
		require.Contains(t, clothing, "variant_products")
		require.NotEqual(t, hardware, clothing)
	})
	t.Run("una plantilla que no existe no tiene pasos", func(t *testing.T) {
		_, ok := steps.Steps("panaderia")
		require.False(t, ok)
	})
}
