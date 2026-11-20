//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/onboarding/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

func TestDismissalRepository(t *testing.T) {
	d := dbtest.New(t)
	repo := postgres.NewDismissalRepository(d)

	t.Run("sin ocultar no hay ocultamiento", func(t *testing.T) {
		tenant, user := seed(t, d, "Ferretería La Tuerca", "ferreteria")

		_, ok, err := repo.Get(t.Context(), tenant, user)
		require.NoError(t, err)
		require.False(t, ok)
	})
	t.Run("guarda el ocultamiento con sus pasos pendientes y lo reemplaza al ocultar otra vez", func(t *testing.T) {
		tenant, user := seed(t, d, "Almacén Paso Firme", "ropa_calzado")
		first := domain.RestoreDismissal(tenant, user, now, []domain.StepID{domain.StepVariantProducts, domain.StepFirstSale})
		require.NoError(t, repo.Save(t.Context(), first))

		later := now.Add(48 * time.Hour)
		require.NoError(t, repo.Save(t.Context(), domain.RestoreDismissal(tenant, user, later, []domain.StepID{domain.StepFirstSale})))

		got, ok, err := repo.Get(t.Context(), tenant, user)
		require.NoError(t, err)
		require.True(t, ok)
		require.True(t, later.Equal(got.DismissedAt()))
		require.Equal(t, []domain.StepID{domain.StepFirstSale}, got.Pending())
		require.Equal(t, user, got.UserID())
	})
	t.Run("guarda un ocultamiento sin pasos pendientes", func(t *testing.T) {
		tenant, user := seed(t, d, "Droguería Central", "drogueria")
		require.NoError(t, repo.Save(t.Context(), domain.RestoreDismissal(tenant, user, now, nil)))

		got, ok, err := repo.Get(t.Context(), tenant, user)
		require.NoError(t, err)
		require.True(t, ok)
		require.Empty(t, got.Pending())
	})
	t.Run("aislamiento: el tenant B no ve ni reemplaza el ocultamiento del A", func(t *testing.T) {
		tenantA, userA := seed(t, d, "Ferretería A", "ferreteria")
		tenantB, _ := seed(t, d, "Almacén B", "ropa_calzado")
		require.NoError(t, repo.Save(t.Context(), domain.RestoreDismissal(tenantA, userA, now, []domain.StepID{domain.StepProducts})))

		_, ok, err := repo.Get(t.Context(), tenantB, userA)
		require.NoError(t, err)
		require.False(t, ok)
		require.NoError(t, repo.Save(t.Context(), domain.RestoreDismissal(tenantB, userA, now.Add(time.Hour), nil)))

		got, ok, err := repo.Get(t.Context(), tenantA, userA)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, []domain.StepID{domain.StepProducts}, got.Pending())
	})
}
