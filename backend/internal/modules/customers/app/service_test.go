package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
)

func TestNewService(t *testing.T) {
	t.Run("falla si falta un repositorio", func(t *testing.T) {
		_, err := app.NewService(app.Deps{Segments: fakeSegments{}})
		require.Error(t, err)
	})
}
