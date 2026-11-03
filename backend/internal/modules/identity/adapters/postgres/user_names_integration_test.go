//go:build integration

package postgres_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUserRepositoryNames(t *testing.T) {
	f := newFixture(t)

	t.Run("devuelve los nombres de los usuarios de la empresa y no los de otra", func(t *testing.T) {
		tenant, other := f.createTenant(t), f.createTenant(t)
		cashier := f.createUser(t, tenant.ID(), "cajera@ferreteria.co")
		stranger := f.createUser(t, other.ID(), "cajero@almacen.co")

		names, err := f.users.Names(t.Context(), tenant.ID(), []uuid.UUID{cashier.ID(), stranger.ID()})
		require.NoError(t, err)
		require.Equal(t, map[uuid.UUID]string{cashier.ID(): "Ana Gómez"}, names)

		names, err = f.users.Names(t.Context(), other.ID(), []uuid.UUID{cashier.ID()})
		require.NoError(t, err)
		require.Empty(t, names)
	})
}
