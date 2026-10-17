//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
)

// clothingSeed is the seed of a clothing store for tenantID, with a unit
// whose code collides with the "und" createTenant already stored when
// collide is true.
func clothingSeed(t *testing.T, tenantID uuid.UUID, collide bool) domain.TenantSeed {
	t.Helper()
	code := "par"
	if collide {
		code = "und"
	}
	pair, err := domain.NewUnit(tenantID, code, "Par", now)
	require.NoError(t, err)
	women, err := domain.NewCategory(tenantID, "Mujer", nil, now)
	require.NoError(t, err)
	gender, err := domain.NewAttributeDefinition(tenantID, domain.AttributeParams{
		Key: "genero", Label: "Género", Type: domain.AttributeList, Options: []string{"Mujer", "Hombre"},
	}, now)
	require.NoError(t, err)
	size, err := domain.NewVariantAxis(tenantID, "talla", []string{"S", "M"}, now)
	require.NoError(t, err)
	return domain.TenantSeed{
		Units: []*domain.UnitOfMeasure{pair}, Categories: []*domain.Category{women},
		Attributes: []*domain.AttributeDefinition{gender}, VariantAxes: []*domain.VariantAxis{size},
	}
}

// seeded counts what tenantID has of each kind.
func (f fixture) seeded(t *testing.T, tenantID uuid.UUID) (units, categories, attributes, axes int) {
	t.Helper()
	u, err := f.units.List(t.Context(), tenantID)
	require.NoError(t, err)
	c, err := f.categories.List(t.Context(), tenantID)
	require.NoError(t, err)
	a, err := f.attributes.List(t.Context(), tenantID)
	require.NoError(t, err)
	x, err := f.axes.List(t.Context(), tenantID)
	require.NoError(t, err)
	return len(u), len(c), len(a), len(x)
}

func TestSeedRepository(t *testing.T) {
	f := newFixture(t)
	seeds := postgres.NewSeedRepository(f.db)

	t.Run("guarda las unidades, categorías, atributos y ejes de la siembra", func(t *testing.T) {
		tenant := f.createTenant(t)

		require.NoError(t, seeds.Seed(t.Context(), tenant, clothingSeed(t, tenant, false)))

		units, categories, attributes, axes := f.seeded(t, tenant)
		require.Equal(t, []int{2, 1, 1, 1}, []int{units, categories, attributes, axes}, "und from createTenant plus par")
	})

	t.Run("si algo falla, no guarda nada", func(t *testing.T) {
		tenant := f.createTenant(t)

		err := seeds.Seed(t.Context(), tenant, clothingSeed(t, tenant, true))

		require.ErrorIs(t, err, domain.ErrUnitCodeTaken)
		units, categories, attributes, axes := f.seeded(t, tenant)
		require.Equal(t, []int{1, 0, 0, 0}, []int{units, categories, attributes, axes})
	})

	t.Run("la siembra de un tenant no se ve desde otro", func(t *testing.T) {
		tenantA, tenantB := f.createTenant(t), f.createTenant(t)

		require.NoError(t, seeds.Seed(t.Context(), tenantA, clothingSeed(t, tenantA, false)))

		units, categories, attributes, axes := f.seeded(t, tenantB)
		require.Equal(t, []int{1, 0, 0, 0}, []int{units, categories, attributes, axes})
	})

	t.Run("no puede sembrar datos de otro tenant", func(t *testing.T) {
		tenantA, tenantB := f.createTenant(t), f.createTenant(t)

		err := seeds.Seed(t.Context(), tenantA, clothingSeed(t, tenantB, false))

		require.Error(t, err, "RLS rejects rows of another tenant")
		units, categories, attributes, axes := f.seeded(t, tenantB)
		require.Equal(t, []int{1, 0, 0, 0}, []int{units, categories, attributes, axes})
	})
}
