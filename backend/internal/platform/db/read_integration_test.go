//go:build integration

package db_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

func TestReadEnLaBase(t *testing.T) {
	t.Run("Read devuelve lo que leyó dentro de la transacción del tenant", func(t *testing.T) {
		d := dbtest.New(t)
		tenantID := uuid.Must(uuid.NewV7())

		got, err := db.Read(t.Context(), d, tenantID, func(tx pgx.Tx) (string, error) {
			return currentTenant(t.Context(), tx)
		})

		require.NoError(t, err)
		require.Equal(t, tenantID.String(), got)
	})

	t.Run("Read devuelve el error de la lectura y el valor cero", func(t *testing.T) {
		d := dbtest.New(t)
		boom := errors.New("boom")

		got, err := db.Read(t.Context(), d, uuid.Must(uuid.NewV7()), func(pgx.Tx) (int, error) { return 7, boom })

		require.ErrorIs(t, err, boom)
		require.Zero(t, got)
	})

	t.Run("Affected devuelve las filas que afectó la escritura", func(t *testing.T) {
		d := dbtest.New(t)

		n, err := db.Affected(t.Context(), d, uuid.Must(uuid.NewV7()), func(tx pgx.Tx) (int64, error) {
			tag, err := tx.Exec(t.Context(), "SELECT 1 UNION ALL SELECT 2")
			return tag.RowsAffected(), err
		})

		require.NoError(t, err)
		require.Equal(t, int64(2), n)
	})
}
