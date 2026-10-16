//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func (f fixture) createAxis(t *testing.T, tenantID uuid.UUID, name string, values ...string) *domain.VariantAxis {
	t.Helper()
	a, err := domain.NewVariantAxis(tenantID, name, values, now)
	require.NoError(t, err)
	require.NoError(t, f.axes.Create(t.Context(), a))
	return a
}

func axisNames(axes []*domain.VariantAxis) []string {
	out := make([]string, 0, len(axes))
	for _, a := range axes {
		out = append(out, a.Name())
	}
	return out
}

func TestVariantAxisRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda, lee, edita y lista los ejes del tenant por nombre", func(t *testing.T) {
		tenant := f.createTenant(t)
		size := f.createAxis(t, tenant, "talla", "S", "M", "L")
		f.createAxis(t, tenant, "color", "Rojo", "Azul")

		got, err := f.axes.GetByID(t.Context(), tenant, size.ID())
		require.NoError(t, err)
		require.Equal(t, tenant, got.TenantID())
		require.Equal(t, "talla", got.Name())
		require.Equal(t, []string{"S", "M", "L"}, got.Values())
		require.Equal(t, now, got.CreatedAt())

		_, err = size.Update([]string{"S", "M", "L", "XL"})
		require.NoError(t, err)
		require.NoError(t, f.axes.Update(t.Context(), size, nil))

		list, err := f.axes.List(t.Context(), tenant)
		require.NoError(t, err)
		require.Equal(t, []string{"color", "talla"}, axisNames(list))
		require.Equal(t, []string{"S", "M", "L", "XL"}, list[1].Values())
	})

	t.Run("un eje inexistente da ErrAxisNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		missing := uuid.Must(uuid.NewV7())

		_, err := f.axes.GetByID(t.Context(), tenant, missing)
		require.ErrorIs(t, err, domain.ErrAxisNotFound)
		require.ErrorIs(t, f.axes.Delete(t.Context(), tenant, missing), domain.ErrAxisNotFound)
		ghost, err := domain.NewVariantAxis(tenant, "talla", []string{"M"}, now)
		require.NoError(t, err)
		require.ErrorIs(t, f.axes.Update(t.Context(), ghost, nil), domain.ErrAxisNotFound)
	})

	t.Run("rechaza dos ejes con el mismo nombre en el tenant", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createAxis(t, tenant, "presentación", "Galón")

		dup, err := domain.NewVariantAxis(tenant, "Presentación", []string{"Cuarto"}, now)
		require.NoError(t, err)
		require.ErrorIs(t, f.axes.Create(t.Context(), dup), domain.ErrAxisNameTaken)
	})

	t.Run("no se puede borrar un eje que usa una variante", func(t *testing.T) {
		tenant := f.createTenant(t)
		size := f.createAxis(t, tenant, "talla", "S", "M")
		unused := f.createAxis(t, tenant, "color", "Rojo")
		f.createProduct(t, tenant, "CAM-S", "Camiseta", withAxes(map[string]string{"talla": "S"}))

		require.ErrorIs(t, f.axes.Delete(t.Context(), tenant, size.ID()), domain.ErrAxisInUse)

		require.NoError(t, f.axes.Delete(t.Context(), tenant, unused.ID()))
		_, err := f.axes.GetByID(t.Context(), tenant, unused.ID())
		require.ErrorIs(t, err, domain.ErrAxisNotFound)
	})

	t.Run("no se puede quitar un valor que usa una variante, aunque esté guardado con otras mayúsculas", func(t *testing.T) {
		tenant := f.createTenant(t)
		size := f.createAxis(t, tenant, "talla", "S", "M", "L")
		// A variant stored before the tenant defined its axes keeps its own spelling.
		p := f.createProduct(t, tenant, "CAM-S", "Camiseta", withAxes(map[string]string{"talla": "s"}))
		f.addVariant(t, p, variant("CAM-M", "", map[string]string{"talla": "M"}))

		removed, err := size.Update([]string{"M", "L"})
		require.NoError(t, err)
		require.ErrorIs(t, f.axes.Update(t.Context(), size, removed), domain.ErrAxisValueInUse)

		stored, err := f.axes.GetByID(t.Context(), tenant, size.ID())
		require.NoError(t, err)
		require.Equal(t, []string{"S", "M", "L"}, stored.Values(), "the update rolled back")

		removed, err = stored.Update([]string{"S", "M"})
		require.NoError(t, err)
		require.NoError(t, f.axes.Update(t.Context(), stored, removed), "quitar un valor sin uso sí se puede")
	})
}

func TestVariantAxisTenantIsolation(t *testing.T) {
	f := newFixture(t)
	tenantA, tenantB := f.createTenant(t), f.createTenant(t)
	sizeA := f.createAxis(t, tenantA, "talla", "S", "M")

	t.Run("dos tenants pueden usar el mismo nombre", func(t *testing.T) {
		f.createAxis(t, tenantB, "talla", "M")
	})

	t.Run("el tenant B no ve, no edita ni borra los ejes del tenant A", func(t *testing.T) {
		_, err := f.axes.GetByID(t.Context(), tenantB, sizeA.ID())
		require.ErrorIs(t, err, domain.ErrAxisNotFound)

		list, err := f.axes.List(t.Context(), tenantB)
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, tenantB, list[0].TenantID())

		require.ErrorIs(t, f.axes.Delete(t.Context(), tenantB, sizeA.ID()), domain.ErrAxisNotFound)
		_, err = f.axes.GetByID(t.Context(), tenantA, sizeA.ID())
		require.NoError(t, err)
	})

	t.Run("las variantes del tenant B no cuentan como uso de los ejes del tenant A", func(t *testing.T) {
		f.createProduct(t, tenantB, "CAM-S", "Camiseta", withAxes(map[string]string{"talla": "S"}))

		removed, err := sizeA.Update([]string{"M"})
		require.NoError(t, err)
		require.NoError(t, f.axes.Update(t.Context(), sizeA, removed))
		require.NoError(t, f.axes.Delete(t.Context(), tenantA, sizeA.ID()))
	})
}
