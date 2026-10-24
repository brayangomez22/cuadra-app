package db_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

func TestRead(t *testing.T) {
	t.Run("sin tenant no abre la transacción y devuelve el valor cero", func(t *testing.T) {
		called := false

		got, err := db.Read(t.Context(), new(db.DB), uuid.Nil, func(pgx.Tx) (string, error) {
			called = true
			return "x", nil
		})

		require.ErrorIs(t, err, db.ErrNoTenant)
		require.Empty(t, got)
		require.False(t, called)
	})

	t.Run("Affected sin tenant devuelve cero filas", func(t *testing.T) {
		n, err := db.Affected(t.Context(), new(db.DB), uuid.Nil, func(pgx.Tx) (int64, error) { return 1, nil })

		require.ErrorIs(t, err, db.ErrNoTenant)
		require.Zero(t, n)
	})
}
