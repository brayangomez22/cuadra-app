//go:build integration

package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTenantRepositoryAnalyticsConsent(t *testing.T) {
	f := newFixture(t)

	t.Run("una empresa nueva no da su consentimiento a la analítica; el consentimiento se guarda y se retira", func(t *testing.T) {
		tenant := f.createTenant(t)
		got, err := f.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.False(t, got.AnalyticsConsent())

		got.SetAnalyticsConsent(true)
		require.NoError(t, f.tenants.Update(t.Context(), got))
		consented, err := f.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.True(t, consented.AnalyticsConsent())

		consented.SetAnalyticsConsent(false)
		require.NoError(t, f.tenants.Update(t.Context(), consented))
		withdrawn, err := f.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.False(t, withdrawn.AnalyticsConsent())
	})
}
