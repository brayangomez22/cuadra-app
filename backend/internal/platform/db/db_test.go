package db_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

func TestWithTenantTxSinTenant(t *testing.T) {
	t.Run("WithTenantTx rechaza un tenant vacío sin abrir la transacción", func(t *testing.T) {
		called := false

		err := new(db.DB).WithTenantTx(t.Context(), uuid.Nil, func(pgx.Tx) error {
			called = true
			return nil
		})

		require.ErrorIs(t, err, db.ErrNoTenant)
		require.False(t, called)
	})
}
