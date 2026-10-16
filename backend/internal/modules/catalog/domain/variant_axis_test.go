package domain_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func mustAxis(t *testing.T, tenantID uuid.UUID, name string, values ...string) *domain.VariantAxis {
	t.Helper()
	a, err := domain.NewVariantAxis(tenantID, name, values, testNow)
	require.NoError(t, err)
	return a
}

func TestNewVariantAxis(t *testing.T) {
	t.Run("crea un eje con el nombre en minúsculas y los valores en su orden", func(t *testing.T) {
		tests := []struct {
			name, axisName string
			values         []string
			wantName       string
			wantValues     []string
		}{
			{"almacén de ropa: talla", " Talla ", []string{"S", " M ", "L"}, "talla", []string{"S", "M", "L"}},
			{"ferretería: presentación", "Presentación", []string{"1/4 galón", "Galón"}, "presentación", []string{"1/4 galón", "Galón"}},
			{"droguería: concentración", "concentración", []string{"500 mg"}, "concentración", []string{"500 mg"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tenantID := uuid.Must(uuid.NewV7())

				a, err := domain.NewVariantAxis(tenantID, tt.axisName, tt.values, testNow)
				require.NoError(t, err)

				require.Equal(t, 7, int(a.ID().Version()))
				require.Equal(t, tenantID, a.TenantID())
				require.Equal(t, tt.wantName, a.Name())
				require.Equal(t, tt.wantValues, a.Values())
				require.Equal(t, testNow.UTC(), a.CreatedAt())
			})
		}
	})

	t.Run("rechaza un nombre vacío o de más de 50 caracteres", func(t *testing.T) {
		for _, name := range []string{"", "  ", strings.Repeat("a", 51)} {
			_, err := domain.NewVariantAxis(uuid.Must(uuid.NewV7()), name, []string{"M"}, testNow)
			require.ErrorIs(t, err, domain.ErrInvalidAxisName, "%q", name)
		}
	})

	t.Run("rechaza un eje sin valores, con valores vacíos o largos", func(t *testing.T) {
		for _, values := range [][]string{nil, {"M", " "}, {strings.Repeat("a", 101)}} {
			_, err := domain.NewVariantAxis(uuid.Must(uuid.NewV7()), "talla", values, testNow)
			require.ErrorIs(t, err, domain.ErrInvalidAxisAllowedValues, "%q", values)
		}
	})

	t.Run("rechaza valores repetidos en un eje sin importar mayúsculas", func(t *testing.T) {
		_, err := domain.NewVariantAxis(uuid.Must(uuid.NewV7()), "talla", []string{"M", " m "}, testNow)
		require.ErrorIs(t, err, domain.ErrInvalidAxisAllowedValues)
	})

	t.Run("rechaza un tenant vacío", func(t *testing.T) {
		_, err := domain.NewVariantAxis(uuid.Nil, "talla", []string{"M"}, testNow)
		require.ErrorIs(t, err, domain.ErrInvalidTenantID)
	})
}

func TestVariantAxisUpdate(t *testing.T) {
	t.Run("cambia los valores y dice cuáles quitó", func(t *testing.T) {
		a := mustAxis(t, uuid.Must(uuid.NewV7()), "talla", "S", "M", "L")

		removed, err := a.Update([]string{"M", "L", "XL"})
		require.NoError(t, err)

		require.Equal(t, []string{"S"}, removed)
		require.Equal(t, []string{"M", "L", "XL"}, a.Values())
		require.Equal(t, "talla", a.Name())
	})

	t.Run("si la edición falla, el eje queda igual", func(t *testing.T) {
		a := mustAxis(t, uuid.Must(uuid.NewV7()), "color", "Rojo")

		_, err := a.Update(nil)
		require.ErrorIs(t, err, domain.ErrInvalidAxisAllowedValues)
		require.Equal(t, []string{"Rojo"}, a.Values())
	})
}

func TestResolveAxisValues(t *testing.T) {
	tenantID := uuid.Must(uuid.NewV7())
	axes := []*domain.VariantAxis{
		mustAxis(t, tenantID, "talla", "S", "M", "L"),
		mustAxis(t, tenantID, "color", "Rojo", "Azul"),
		mustAxis(t, tenantID, "presentación", "1/4 galón", "Galón"),
	}

	t.Run("escribe cada valor como lo define su eje", func(t *testing.T) {
		tests := []struct {
			name string
			in   map[string]string
			want map[string]string
		}{
			{"almacén de ropa: acepta 'm' para el valor 'M' y lo guarda como 'M'",
				map[string]string{"Talla": " m ", "color": "ROJO"}, map[string]string{"talla": "M", "color": "Rojo"}},
			{"ferretería: presentación", map[string]string{"presentación": "galón"}, map[string]string{"presentación": "Galón"}},
			{"producto simple, sin ejes", map[string]string{}, map[string]string{}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := domain.ResolveAxisValues(axes, tt.in)
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("rechaza un eje que el tenant no definió", func(t *testing.T) {
		_, err := domain.ResolveAxisValues(axes, map[string]string{"talla": "M", "material": "Algodón"})
		require.ErrorIs(t, err, domain.ErrUnknownAxis)
		var axisErr *domain.AxisError
		require.ErrorAs(t, err, &axisErr)
		require.Equal(t, "material", axisErr.Axis)
	})

	t.Run("rechaza un valor que el eje no permite", func(t *testing.T) {
		_, err := domain.ResolveAxisValues(axes, map[string]string{"talla": "XL"})
		require.ErrorIs(t, err, domain.ErrAxisValueNotAllowed)
		var axisErr *domain.AxisError
		require.ErrorAs(t, err, &axisErr)
		require.Equal(t, "talla", axisErr.Axis)
	})

	t.Run("rechaza dos nombres que son el mismo eje", func(t *testing.T) {
		_, err := domain.ResolveAxisValues(axes, map[string]string{"talla": "M", "Talla": "L"})
		require.ErrorIs(t, err, domain.ErrInvalidAxisValues)
	})
}
