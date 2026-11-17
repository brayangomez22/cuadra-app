package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/auth"
)

func TestJWTRotation(t *testing.T) {
	oldSecret := []byte(strings.Repeat("o", 32))
	newSecret := []byte(strings.Repeat("n", 32))
	tenantID, userID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	before, err := auth.NewJWT(oldSecret, 15*time.Minute)
	require.NoError(t, err)
	issued, _, err := before.Issue(tenantID, userID, "cashier")
	require.NoError(t, err)

	t.Run("con la llave anterior, un token firmado antes de rotar sigue válido", func(t *testing.T) {
		after, err := auth.NewJWT(newSecret, 15*time.Minute, auth.WithPreviousSecret(oldSecret))
		require.NoError(t, err)
		p, err := after.Verify(issued)
		require.NoError(t, err)
		require.Equal(t, userID, p.UserID)
	})

	t.Run("los tokens nuevos se firman solo con la llave actual", func(t *testing.T) {
		after, err := auth.NewJWT(newSecret, 15*time.Minute, auth.WithPreviousSecret(oldSecret))
		require.NoError(t, err)
		fresh, _, err := after.Issue(tenantID, userID, "cashier")
		require.NoError(t, err)
		_, err = before.Verify(fresh)
		require.ErrorIs(t, err, auth.ErrInvalidToken)
	})

	t.Run("sin la llave anterior, un token viejo ya no sirve", func(t *testing.T) {
		after, err := auth.NewJWT(newSecret, 15*time.Minute)
		require.NoError(t, err)
		_, err = after.Verify(issued)
		require.ErrorIs(t, err, auth.ErrInvalidToken)
	})

	t.Run("rechaza una llave anterior débil", func(t *testing.T) {
		_, err := auth.NewJWT(newSecret, 15*time.Minute, auth.WithPreviousSecret([]byte("corta")))
		require.ErrorIs(t, err, auth.ErrWeakSecret)
	})
}
