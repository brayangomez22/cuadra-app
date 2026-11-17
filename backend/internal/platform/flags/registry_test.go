package flags_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/flags"
)

func TestNewRegistry(t *testing.T) {
	cases := []struct {
		name string
		defs []flags.Definition
		err  string
	}{
		{"rechaza una llave que no es snake_case", []flags.Definition{{Key: "NuevoPOS", Description: "x"}}, "snake_case"},
		{"rechaza una flag sin descripción", []flags.Definition{{Key: "new_pos"}}, "description"},
		{"rechaza una flag repetida", []flags.Definition{{Key: "new_pos", Description: "x"}, {Key: "new_pos", Description: "y"}}, "twice"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := flags.NewRegistry(tc.defs...)
			require.ErrorContains(t, err, tc.err)
		})
	}

	t.Run("las flags del producto son válidas", func(t *testing.T) {
		r, err := flags.NewRegistry(flags.Builtin()...)
		require.NoError(t, err)
		require.True(t, r.Known("analytics"))
		require.False(t, r.Known("nope"))
	})
}
