//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestRefreshTokenPurge(t *testing.T) {
	f := newTokenFixture(t)
	afterExpiry := now.Add(domain.RefreshTokenTTL + time.Minute)

	t.Run("encuentra las empresas con tokens vencidos y borra solo los de cada una", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		expiredA := f.createToken(t, a)
		f.createToken(t, b)

		tenants, err := f.tokens.TenantsWithExpired(t.Context(), afterExpiry, 10_000)
		require.NoError(t, err)
		require.Contains(t, tenants, a.ID())
		require.Contains(t, tenants, b.ID())

		n, err := f.tokens.DeleteExpired(t.Context(), a.ID(), afterExpiry)
		require.NoError(t, err)
		require.EqualValues(t, 1, n)
		_, err = f.tokens.GetByHash(t.Context(), a.ID(), expiredA.Hash())
		require.ErrorIs(t, err, domain.ErrRefreshTokenNotFound)

		tenants, err = f.tokens.TenantsWithExpired(t.Context(), afterExpiry, 10_000)
		require.NoError(t, err)
		require.NotContains(t, tenants, a.ID())
		require.Contains(t, tenants, b.ID(), "aislamiento: borrar los de A no toca los de B")
	})

	t.Run("no borra los tokens vigentes", func(t *testing.T) {
		tn := f.createTenant(t)
		live := f.createToken(t, tn)

		n, err := f.tokens.DeleteExpired(t.Context(), tn.ID(), now)
		require.NoError(t, err)
		require.Zero(t, n)
		_, err = f.tokens.GetByHash(t.Context(), tn.ID(), live.Hash())
		require.NoError(t, err)
	})
}
