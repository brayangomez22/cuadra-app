package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/postgres"
)

func TestTokenCipherRotation(t *testing.T) {
	tenant := uuid.Must(uuid.NewV7())
	const token = "muisca-org-7f2c.dG9rZW4tc2VjcmV0bw"
	old, err := postgres.NewTokenCipher(key(1))
	require.NoError(t, err)
	sealed, err := old.Seal(tenant, token)
	require.NoError(t, err)

	t.Run("con la llave anterior abre lo cifrado antes de rotar y avisa que hay que volver a cifrarlo", func(t *testing.T) {
		rotated, err := postgres.NewTokenCipher(key(2), key(1))
		require.NoError(t, err)
		got, stale, err := rotated.OpenRotating(tenant, sealed)
		require.NoError(t, err)
		require.True(t, stale)
		require.Equal(t, token, got)
	})

	t.Run("lo cifrado con la llave actual no está vencido", func(t *testing.T) {
		rotated, err := postgres.NewTokenCipher(key(2), key(1))
		require.NoError(t, err)
		fresh, err := rotated.Seal(tenant, token)
		require.NoError(t, err)
		got, stale, err := rotated.OpenRotating(tenant, fresh)
		require.NoError(t, err)
		require.False(t, stale)
		require.Equal(t, token, got)
		_, err = old.Open(tenant, fresh)
		require.Error(t, err, "cifra solo con la llave actual")
	})

	t.Run("sin la llave anterior no abre lo viejo", func(t *testing.T) {
		rotated, err := postgres.NewTokenCipher(key(2))
		require.NoError(t, err)
		_, _, err = rotated.OpenRotating(tenant, sealed)
		require.Error(t, err)
	})

	t.Run("rechaza una llave anterior de otro tamaño", func(t *testing.T) {
		_, err := postgres.NewTokenCipher(key(2), []byte("corta"))
		require.Error(t, err)
	})
}
