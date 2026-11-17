package analytics_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/analytics"
)

var key = []byte("analytics-key-only-for-tests-0123456789")

func TestPseudonyms(t *testing.T) {
	user := uuid.Must(uuid.NewV7())

	t.Run("el seudónimo de un usuario es estable y no revela su ID", func(t *testing.T) {
		p := analytics.NewPseudonyms(key)

		first, ok := p.Pseudonym(user)
		require.True(t, ok)
		second, _ := p.Pseudonym(user)
		require.Equal(t, first, second)
		require.Len(t, first, 32)
		require.NotContains(t, first, strings.ReplaceAll(user.String(), "-", ""))
	})

	t.Run("dos usuarios tienen seudónimos distintos", func(t *testing.T) {
		p := analytics.NewPseudonyms(key)
		a, _ := p.Pseudonym(user)
		b, _ := p.Pseudonym(uuid.Must(uuid.NewV7()))
		require.NotEqual(t, a, b)
	})

	t.Run("con otra clave el seudónimo cambia: sin la clave no se recalcula", func(t *testing.T) {
		a, _ := analytics.NewPseudonyms(key).Pseudonym(user)
		b, _ := analytics.NewPseudonyms([]byte("another-analytics-key-0123456789abcdef")).Pseudonym(user)
		require.NotEqual(t, a, b)
	})

	t.Run("sin clave la analítica está apagada", func(t *testing.T) {
		id, ok := analytics.NewPseudonyms(nil).Pseudonym(user)
		require.False(t, ok)
		require.Empty(t, id)
	})
}
