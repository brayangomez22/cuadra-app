//go:build integration

package postgres_test

import (
	"bytes"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

const orgToken = "muisca-org-7f2c.dG9rZW4tc2VjcmV0bw"

func account(t *testing.T, mode, testSet, token string) domain.ProviderAccount {
	t.Helper()
	var s domain.Secret
	if token != "" {
		var err error
		s, err = domain.NewSecret(token)
		require.NoError(t, err)
	}
	a, err := domain.NewProviderAccount(mode, testSet, s)
	require.NoError(t, err)
	return a
}

// storedToken reads the token's column as the database keeps it.
func (f fixture) storedToken(t *testing.T, tn *tenant) []byte {
	t.Helper()
	var b []byte
	require.NoError(t, f.db.WithTenantTx(t.Context(), tn.id, func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(), "SELECT token_ciphertext FROM invoicing_provider_accounts WHERE tenant_id = $1", tn.id).Scan(&b)
	}))
	return b
}

func TestProviderAccountRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("una empresa sin cuenta está en pruebas y sin token", func(t *testing.T) {
		tn := f.createTenant(t)
		got, err := f.accounts.Get(t.Context(), tn.id)
		require.NoError(t, err)
		require.Equal(t, domain.DefaultProviderAccount(), got)
	})

	t.Run("guarda el token cifrado y lo devuelve descifrado", func(t *testing.T) {
		tn := f.createTenant(t)
		require.NoError(t, f.accounts.Save(t.Context(), tn.id, account(t, "habilitation", "set-123", orgToken)))

		got, err := f.accounts.Get(t.Context(), tn.id)
		require.NoError(t, err)
		require.Equal(t, domain.ModeHabilitation, got.Mode)
		require.Equal(t, "set-123", got.TestSetID)
		require.Equal(t, orgToken, got.Token.Reveal())
		require.False(t, bytes.Contains(f.storedToken(t, tn), []byte(orgToken)), "the token is stored in clear")
	})

	t.Run("una cuenta sin token se guarda sin token", func(t *testing.T) {
		tn := f.createTenant(t)
		require.NoError(t, f.accounts.Save(t.Context(), tn.id, account(t, "production", "", "")))
		got, err := f.accounts.Get(t.Context(), tn.id)
		require.NoError(t, err)
		require.Equal(t, domain.ModeProduction, got.Mode)
		require.False(t, got.HasToken())
		require.Nil(t, f.storedToken(t, tn))
	})

	t.Run("con otra llave el token no se abre", func(t *testing.T) {
		tn := f.createTenant(t)
		require.NoError(t, f.accounts.Save(t.Context(), tn.id, account(t, "test", "", orgToken)))
		other := postgres.NewProviderAccountRepository(f.db, mustCipher(t, bytes.Repeat([]byte{9}, 32)))
		_, err := other.Get(t.Context(), tn.id)
		require.Error(t, err)
		require.NotContains(t, err.Error(), orgToken)
	})

	t.Run("aislamiento: el tenant B no ve ni cambia la cuenta del tenant A", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		require.NoError(t, f.accounts.Save(t.Context(), a.id, account(t, "production", "", orgToken)))

		got, err := f.accounts.Get(t.Context(), b.id)
		require.NoError(t, err)
		require.False(t, got.HasToken())

		var affected int64
		require.NoError(t, f.db.WithTenantTx(t.Context(), b.id, func(tx pgx.Tx) error {
			tag, err := tx.Exec(t.Context(), "UPDATE invoicing_provider_accounts SET mode = 'test' WHERE tenant_id = $1", a.id)
			affected = tag.RowsAffected()
			return err
		}))
		require.Zero(t, affected)
	})
}
