//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

func TestLocationRepository(t *testing.T) {
	f := newFixture(t)
	tenant := f.createTenant(t)

	t.Run("crea, lee y lista sedes por nombre", func(t *testing.T) {
		tenant := f.createTenant(t)
		north := f.createLocation(t, tenant, "Sede norte")
		f.createLocation(t, tenant, "Bodega")

		got, err := f.locations.GetByID(t.Context(), tenant, north.ID())
		require.NoError(t, err)
		require.Equal(t, "Sede norte", got.Name())
		require.True(t, got.IsActive())
		require.Equal(t, now, got.CreatedAt())

		list, err := f.locations.List(t.Context(), tenant)
		require.NoError(t, err)
		require.Len(t, list, 2)
		require.Equal(t, "Bodega", list[0].Name())
	})

	t.Run("rechaza nombre repetido sin importar mayúsculas", func(t *testing.T) {
		f.createLocation(t, tenant, "Bodega principal")
		l, err := domain.NewLocation(tenant, "BODEGA principal", now)
		require.NoError(t, err)
		require.ErrorIs(t, f.locations.Create(t.Context(), l), domain.ErrLocationNameTaken)
	})

	t.Run("una sede inexistente da ErrLocationNotFound", func(t *testing.T) {
		_, err := f.locations.GetByID(t.Context(), tenant, newID())
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
	})
}

func TestPolicyRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("sin fila de configuración, la política prohíbe stock negativo", func(t *testing.T) {
		tenant := f.createTenant(t)
		policy, err := f.policies.Get(t.Context(), tenant)
		require.NoError(t, err)
		require.False(t, policy.AllowNegative)
	})

	t.Run("lee la política guardada del tenant", func(t *testing.T) {
		tenant := f.createTenant(t)
		f.exec(t, tenant, "INSERT INTO inventory_settings (tenant_id, allow_negative_stock) VALUES ($1, true)", tenant)
		policy, err := f.policies.Get(t.Context(), tenant)
		require.NoError(t, err)
		require.True(t, policy.AllowNegative)
	})
}

func TestTenantIsolation(t *testing.T) {
	f := newFixture(t)
	tenantA := f.createTenant(t)
	tenantB := f.createTenant(t)
	variantA := f.createVariant(t, tenantA)
	variantB := f.createVariant(t, tenantB)
	locA := f.createLocation(t, tenantA, "Bodega")
	locB := f.createLocation(t, tenantB, "Bodega")
	f.receive(t, tenantA, variantA, locA.ID(), "10", "1000")

	t.Run("aislamiento: el tenant B no ve sedes, niveles ni kardex del tenant A", func(t *testing.T) {
		_, err := f.locations.GetByID(t.Context(), tenantB, locA.ID())
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
		list, err := f.locations.List(t.Context(), tenantB)
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, locB.ID(), list[0].ID())

		levels, err := f.stock.ListLevels(t.Context(), tenantB, domain.StockFilter{LocationID: locA.ID(), Limit: 10})
		require.NoError(t, err)
		require.Zero(t, levels.Total)
		kardex, err := f.stock.Kardex(t.Context(), tenantB, domain.KardexFilter{VariantID: variantA, Limit: 10})
		require.NoError(t, err)
		require.Zero(t, kardex.Total)
	})

	t.Run("aislamiento: el tenant B no puede registrar movimientos sobre una variante del tenant A", func(t *testing.T) {
		err := f.stock.Apply(t.Context(), tenantB, variantA, []uuid.UUID{locB.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Receive(dec("1"), dec("1"), info(newID()))
				return []*domain.StockMovement{m}, err
			})
		require.ErrorIs(t, err, domain.ErrVariantNotFound)
	})

	t.Run("aislamiento: el tenant B no puede mover stock en una sede del tenant A", func(t *testing.T) {
		err := f.stock.Apply(t.Context(), tenantB, variantB, []uuid.UUID{locA.ID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Receive(dec("1"), dec("1"), info(newID()))
				return []*domain.StockMovement{m}, err
			})
		require.ErrorIs(t, err, domain.ErrLocationNotFound)
		require.True(t, dec("10").Equal(f.level(t, tenantA, variantA, locA.ID())))
	})

	t.Run("aislamiento: un UPDATE desde el tenant B no cambia los niveles del tenant A", func(t *testing.T) {
		f.exec(t, tenantB, "UPDATE stock_levels SET quantity = 0")
		require.True(t, dec("10").Equal(f.level(t, tenantA, variantA, locA.ID())))
	})
}
