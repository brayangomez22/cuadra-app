package postgres_test

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/postgres"
)

func key(b byte) []byte { return bytes.Repeat([]byte{b}, 32) }

func TestTokenCipher(t *testing.T) {
	tenant := uuid.Must(uuid.NewV7())
	const token = "muisca-org-7f2c.dG9rZW4tc2VjcmV0bw"

	t.Run("cifra y descifra el token del tenant", func(t *testing.T) {
		c, err := postgres.NewTokenCipher(key(1))
		require.NoError(t, err)
		sealed, err := c.Seal(tenant, token)
		require.NoError(t, err)
		require.NotContains(t, string(sealed), token)

		got, err := c.Open(tenant, sealed)
		require.NoError(t, err)
		require.Equal(t, token, got)
	})

	t.Run("cifrar dos veces da textos distintos", func(t *testing.T) {
		c, err := postgres.NewTokenCipher(key(1))
		require.NoError(t, err)
		a, err := c.Seal(tenant, token)
		require.NoError(t, err)
		b, err := c.Seal(tenant, token)
		require.NoError(t, err)
		require.NotEqual(t, a, b)
	})

	t.Run("no se descifra con otra llave", func(t *testing.T) {
		c, err := postgres.NewTokenCipher(key(1))
		require.NoError(t, err)
		sealed, err := c.Seal(tenant, token)
		require.NoError(t, err)
		other, err := postgres.NewTokenCipher(key(2))
		require.NoError(t, err)
		_, err = other.Open(tenant, sealed)
		require.Error(t, err)
	})

	t.Run("el token de un tenant no se descifra como el de otro", func(t *testing.T) {
		c, err := postgres.NewTokenCipher(key(1))
		require.NoError(t, err)
		sealed, err := c.Seal(tenant, token)
		require.NoError(t, err)
		_, err = c.Open(uuid.Must(uuid.NewV7()), sealed)
		require.Error(t, err)
	})

	t.Run("un texto cifrado alterado se rechaza", func(t *testing.T) {
		c, err := postgres.NewTokenCipher(key(1))
		require.NoError(t, err)
		sealed, err := c.Seal(tenant, token)
		require.NoError(t, err)
		sealed[len(sealed)-1] ^= 1
		_, err = c.Open(tenant, sealed)
		require.Error(t, err)
		_, err = c.Open(tenant, sealed[:5])
		require.Error(t, err)
	})

	t.Run("exige una llave de 32 bytes", func(t *testing.T) {
		_, err := postgres.NewTokenCipher(key(1)[:16])
		require.Error(t, err)
	})
}
