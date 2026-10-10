package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func mustNIT(t *testing.T, s string) domain.NIT {
	t.Helper()
	nit, err := domain.ParseNIT(s)
	require.NoError(t, err)
	return nit
}

func TestNewTenant(t *testing.T) {
	now := time.Date(2026, 10, 9, 15, 0, 0, 0, time.FixedZone("COT", -5*3600))

	t.Run("crea tenant activo con id v7", func(t *testing.T) {
		tenant, err := domain.NewTenant("  Ferretería El Tornillo  ", mustNIT(t, "890903938-8"), now)
		require.NoError(t, err)
		require.Equal(t, 7, int(tenant.ID().Version()))
		require.Equal(t, "Ferretería El Tornillo", tenant.Name())
		require.Equal(t, "890903938-8", tenant.NIT().String())
		require.Equal(t, domain.TenantActive, tenant.Status())
		require.True(t, tenant.IsActive())
		require.Equal(t, now.UTC(), tenant.CreatedAt())
		require.Equal(t, time.UTC, tenant.CreatedAt().Location())
	})

	t.Run("rechaza tenant sin nombre o con nombre demasiado largo", func(t *testing.T) {
		for _, name := range []string{"", "   ", strings.Repeat("a", 201)} {
			_, err := domain.NewTenant(name, mustNIT(t, "890903938-8"), now)
			require.ErrorIs(t, err, domain.ErrInvalidTenantName)
		}
	})

	t.Run("rechaza tenant sin NIT", func(t *testing.T) {
		_, err := domain.NewTenant("Ferretería El Tornillo", domain.NIT{}, now)
		require.ErrorIs(t, err, domain.ErrInvalidNIT)
	})

	t.Run("suspende y reactiva un tenant", func(t *testing.T) {
		tenant, err := domain.NewTenant("Ferretería El Tornillo", mustNIT(t, "890903938-8"), now)
		require.NoError(t, err)

		tenant.Suspend()
		require.Equal(t, domain.TenantSuspended, tenant.Status())
		require.False(t, tenant.IsActive())

		tenant.Activate()
		require.True(t, tenant.IsActive())
	})
}
