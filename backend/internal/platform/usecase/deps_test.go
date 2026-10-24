package usecase_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

type matrix struct{}

func TestAnyNil(t *testing.T) {
	var noReader io.Reader
	var noMatrix *matrix
	var noFunc func()

	cases := []struct {
		name string
		deps []any
		want bool
	}{
		{"todas las dependencias están", []any{io.Discard, &matrix{}, func() {}}, false},
		{"falta una interfaz", []any{io.Discard, noReader}, true},
		{"falta un puntero, aunque llegue dentro de una interfaz", []any{io.Discard, noMatrix}, true},
		{"falta una función", []any{noFunc}, true},
		{"sin dependencias", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, usecase.AnyNil(c.deps...))
		})
	}
}
