package argon2id_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/adapters/argon2id"
)

// cheap keeps the tests fast; production uses argon2id.DefaultParams.
var cheap = argon2id.Params{Memory: 64, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}

func TestHasher(t *testing.T) {
	h := argon2id.New(cheap)

	t.Run("la contraseña se verifica contra su hash", func(t *testing.T) {
		hash, err := h.Hash("cemento-gris-50kg")
		require.NoError(t, err)

		ok, err := h.Verify("cemento-gris-50kg", hash)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("rechaza contraseña incorrecta", func(t *testing.T) {
		hash, err := h.Hash("cemento-gris-50kg")
		require.NoError(t, err)

		ok, err := h.Verify("cemento-gris-50kG", hash)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("dos hashes de la misma contraseña son distintos por el salt", func(t *testing.T) {
		a, err := h.Hash("cemento-gris-50kg")
		require.NoError(t, err)
		b, err := h.Hash("cemento-gris-50kg")
		require.NoError(t, err)
		require.NotEqual(t, a, b)
	})

	t.Run("el hash usa formato argon2id PHC con sus parámetros", func(t *testing.T) {
		hash, err := argon2id.New(argon2id.DefaultParams).Hash("cemento-gris-50kg")
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$"), hash)
		require.Len(t, strings.Split(hash, "$"), 6)
		require.NotContains(t, hash, "cemento")
	})

	t.Run("verifica con los parámetros guardados en el hash y no con los del hasher", func(t *testing.T) {
		hash, err := h.Hash("cemento-gris-50kg")
		require.NoError(t, err)

		stronger := argon2id.New(argon2id.Params{Memory: 128, Iterations: 2, Parallelism: 1, SaltLength: 16, KeyLength: 32})
		ok, err := stronger.Verify("cemento-gris-50kg", hash)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("devuelve error con hash malformado", func(t *testing.T) {
		for _, hash := range []string{
			"",
			"texto-plano",
			"$2a$10$abcdefghijklmnopqrstuv", // bcrypt
			"$argon2i$v=19$m=64,t=1,p=1$c2FsdHNhbHRzYWx0$aGFzaA",                      // argon2i, not argon2id
			"$argon2id$v=16$m=64,t=1,p=1$c2FsdHNhbHRzYWx0$aGFzaA",                     // old version
			"$argon2id$v=19$m=64,t=0,p=1$c2FsdHNhbHRzYWx0$aGFzaA",                     // zero iterations
			"$argon2id$v=19$m=64,t=1,p=0$c2FsdHNhbHRzYWx0$aGFzaA",                     // zero parallelism
			"$argon2id$v=19$m=64,t=1,p=1$no-es-base64!$aGFzaA",                        // bad salt
			"$argon2id$v=19$m=64,t=1,p=1$c2FsdHNhbHRzYWx0$",                           // empty key
			"$argon2id$v=19$m=abc,t=1,p=1$c2FsdHNhbHRzYWx0$aGFzaA",                    // bad params
			"$argon2id$v=19$m=64,t=1,p=1x$c2FsdHNhbHRzYWx0$aGFzaA",                    // trailing garbage
			"$argon2id$v=19$m=64,t=1,p=1$c2FsdHNhbHRzYWx0$" + strings.Repeat("A", 88), // key longer than 64 bytes
		} {
			ok, err := h.Verify("cemento-gris-50kg", hash)
			require.ErrorIs(t, err, argon2id.ErrMalformedHash, hash)
			require.False(t, ok)
		}
	})
}
