package db_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

func TestPageBounds(t *testing.T) {
	t.Run("convierte una página válida a int32", func(t *testing.T) {
		limit, offset, err := db.PageBounds(20, 40)

		require.NoError(t, err)
		require.Equal(t, int32(20), limit)
		require.Equal(t, int32(40), offset)
	})

	for _, c := range []struct {
		name          string
		limit, offset int
	}{
		{"rechaza un límite negativo", -1, 0},
		{"rechaza un desplazamiento negativo", 10, -1},
		{"rechaza un límite que no cabe en int32", math.MaxInt32 + 1, 0},
		{"rechaza un desplazamiento que no cabe en int32", 10, math.MaxInt32 + 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := db.PageBounds(c.limit, c.offset)

			require.ErrorIs(t, err, db.ErrPageOutOfRange)
		})
	}
}
