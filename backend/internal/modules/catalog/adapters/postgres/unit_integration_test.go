//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

func unitNames(units []*domain.UnitOfMeasure) []string {
	out := make([]string, 0, len(units))
	for _, u := range units {
		out = append(out, u.Name())
	}
	return out
}

func TestUnitRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda, lee, edita y lista las unidades del tenant por nombre", func(t *testing.T) {
		tenant := f.createTenant(t)
		pair := f.createUnit(t, tenant, "par", "Par")
		bulto := f.createUnit(t, tenant, "bulto", "Bulto")

		got, err := f.units.GetByID(t.Context(), tenant, pair.ID())
		require.NoError(t, err)
		require.Equal(t, pair.Ref(), got.Ref())
		require.Equal(t, tenant, got.TenantID())
		require.Equal(t, now, got.CreatedAt())

		got, err = f.units.GetByCode(t.Context(), tenant, "bulto")
		require.NoError(t, err)
		require.Equal(t, bulto.ID(), got.ID())

		require.NoError(t, pair.Update("pares", "Pares"))
		require.NoError(t, f.units.Update(t.Context(), pair))

		list, err := f.units.List(t.Context(), tenant)
		require.NoError(t, err)
		require.Equal(t, []string{"Bulto", "Pares", "Unidad"}, unitNames(list))
	})

	t.Run("una unidad inexistente da ErrUnitNotFound", func(t *testing.T) {
		tenant := f.createTenant(t)
		missing := uuid.Must(uuid.NewV7())

		_, err := f.units.GetByID(t.Context(), tenant, missing)
		require.ErrorIs(t, err, domain.ErrUnitNotFound)
		_, err = f.units.GetByCode(t.Context(), tenant, "docena")
		require.ErrorIs(t, err, domain.ErrUnitNotFound)
		require.ErrorIs(t, f.units.Delete(t.Context(), tenant, missing), domain.ErrUnitNotFound)
		ghost, err := domain.NewUnit(tenant, "docena", "Docena", now)
		require.NoError(t, err)
		require.ErrorIs(t, f.units.Update(t.Context(), ghost), domain.ErrUnitNotFound)
	})

	t.Run("rechaza dos unidades con el mismo código en el tenant", func(t *testing.T) {
		tenant := f.createTenant(t)
		dup, err := domain.NewUnit(tenant, "und", "Unidades", now)
		require.NoError(t, err)
		require.ErrorIs(t, f.units.Create(t.Context(), dup), domain.ErrUnitCodeTaken)

		kilo := f.createUnit(t, tenant, "kg", "Kilogramo")
		require.NoError(t, kilo.Update("und", "Kilogramo"))
		require.ErrorIs(t, f.units.Update(t.Context(), kilo), domain.ErrUnitCodeTaken)
	})

	t.Run("CreateMany guarda todas las unidades o ninguna", func(t *testing.T) {
		tenant := f.createTenant(t)
		var batch []*domain.UnitOfMeasure
		for _, c := range []struct{ code, name string }{{"m", "Metro"}, {"kg", "Kilogramo"}, {"und", "Repetida"}} {
			u, err := domain.NewUnit(tenant, c.code, c.name, now)
			require.NoError(t, err)
			batch = append(batch, u)
		}

		require.ErrorIs(t, f.units.CreateMany(t.Context(), tenant, batch), domain.ErrUnitCodeTaken)
		list, err := f.units.List(t.Context(), tenant)
		require.NoError(t, err)
		require.Equal(t, []string{"Unidad"}, unitNames(list))

		require.NoError(t, f.units.CreateMany(t.Context(), tenant, batch[:2]))
		list, err = f.units.List(t.Context(), tenant)
		require.NoError(t, err)
		require.Equal(t, []string{"Kilogramo", "Metro", "Unidad"}, unitNames(list))
	})

	t.Run("no se puede borrar una unidad en uso", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.createProduct(t, tenant, "CAM-S", "Camiseta básica")

		require.ErrorIs(t, f.units.Delete(t.Context(), tenant, f.piece[tenant].ID), domain.ErrUnitInUse)

		unused := f.createUnit(t, tenant, "par", "Par")
		require.NoError(t, f.units.Delete(t.Context(), tenant, unused.ID()))
		_, err := f.units.GetByID(t.Context(), tenant, unused.ID())
		require.ErrorIs(t, err, domain.ErrUnitNotFound)
	})

	t.Run("un producto no puede usar la unidad de otro tenant", func(t *testing.T) {
		tenantA, tenantB := f.createTenant(t), f.createTenant(t)
		p := f.newProduct(t, tenantA, "CEM-50", "Cemento gris 50kg")
		require.NoError(t, p.UpdateDetails(domain.ProductDetails{
			Name: p.Name(), BaseUnit: f.piece[tenantB], TaxRate: p.TaxRate(),
		}))

		require.ErrorIs(t, f.products.Create(t.Context(), p), domain.ErrUnitNotFound)

		stored := f.createProduct(t, tenantA, "CEM-25", "Cemento gris 25kg")
		require.NoError(t, stored.UpdateDetails(domain.ProductDetails{
			Name: stored.Name(), BaseUnit: f.piece[tenantB], TaxRate: stored.TaxRate(),
		}))
		require.ErrorIs(t, f.products.Update(t.Context(), stored), domain.ErrUnitNotFound)
	})
}

func TestUnitTenantIsolation(t *testing.T) {
	f := newFixture(t)
	tenantA, tenantB := f.createTenant(t), f.createTenant(t)
	unitA := f.createUnit(t, tenantA, "par", "Par")

	t.Run("dos tenants pueden usar el mismo código", func(t *testing.T) {
		f.createUnit(t, tenantB, "par", "Par")
	})

	t.Run("el tenant B no ve las unidades del tenant A", func(t *testing.T) {
		_, err := f.units.GetByID(t.Context(), tenantB, unitA.ID())
		require.ErrorIs(t, err, domain.ErrUnitNotFound)

		list, err := f.units.List(t.Context(), tenantB)
		require.NoError(t, err)
		for _, u := range list {
			require.Equal(t, tenantB, u.TenantID())
			require.NotEqual(t, unitA.ID(), u.ID())
		}
	})

	t.Run("el tenant B no edita ni borra las unidades del tenant A", func(t *testing.T) {
		forged, err := domain.RehydrateUnit(domain.UnitSnapshot{
			ID: unitA.ID(), TenantID: tenantB, Code: "hack", Name: "Hack", CreatedAt: now,
		})
		require.NoError(t, err)
		require.ErrorIs(t, f.units.Update(t.Context(), forged), domain.ErrUnitNotFound)
		require.ErrorIs(t, f.units.Delete(t.Context(), tenantB, unitA.ID()), domain.ErrUnitNotFound)

		got, err := f.units.GetByID(t.Context(), tenantA, unitA.ID())
		require.NoError(t, err)
		require.Equal(t, "par", got.Code())
		require.Equal(t, "Par", got.Name())
	})
}
