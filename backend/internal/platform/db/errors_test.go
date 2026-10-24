package db_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

func TestViolates(t *testing.T) {
	unique := &pgconn.PgError{Code: db.UniqueViolation, ConstraintName: "units_code_key"}
	foreignKey := &pgconn.PgError{Code: db.ForeignKeyViolation, ConstraintName: "products_category_fkey"}

	cases := []struct {
		name       string
		err        error
		code       string
		constraint string
		want       bool
	}{
		{"la restricción y el código coinciden", unique, db.UniqueViolation, "units_code_key", true},
		{"coincide aunque el error venga envuelto", fmt.Errorf("catalog: %w", unique), db.UniqueViolation, "units_code_key", true},
		{"otra restricción del mismo código", unique, db.UniqueViolation, "sku_key", false},
		{"la misma restricción con otro código", foreignKey, db.UniqueViolation, "products_category_fkey", false},
		{"un error que no es de Postgres", errors.New("boom"), db.UniqueViolation, "units_code_key", false},
		{"sin error", nil, db.UniqueViolation, "units_code_key", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, db.Violates(c.err, c.code, c.constraint))
		})
	}

	t.Run("HasCode mira solo el código, de cualquier restricción", func(t *testing.T) {
		require.True(t, db.HasCode(foreignKey, db.ForeignKeyViolation))
		require.False(t, db.HasCode(unique, db.ForeignKeyViolation))
		require.False(t, db.HasCode(errors.New("boom"), db.ForeignKeyViolation))
	})
}
