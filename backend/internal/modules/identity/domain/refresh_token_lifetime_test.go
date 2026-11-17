package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestSessionMaxAge(t *testing.T) {
	login := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	tenantID, userID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	t.Run("el login empieza la familia y cada rotación la hereda", func(t *testing.T) {
		token, _, err := domain.NewRefreshToken(tenantID, userID, login)
		require.NoError(t, err)
		require.Equal(t, login, token.FamilyStartedAt())

		next, _, err := token.Rotate(login.Add(24 * time.Hour))
		require.NoError(t, err)
		require.Equal(t, login, next.FamilyStartedAt())
		require.Equal(t, login.Add(48*time.Hour).Add(domain.RefreshTokenTTL-24*time.Hour), next.ExpiresAt())
	})

	t.Run("rotar a diario no extiende la sesión más allá de su vida máxima", func(t *testing.T) {
		token, _, err := domain.NewRefreshToken(tenantID, userID, login)
		require.NoError(t, err)
		end := login.Add(domain.SessionMaxAge)
		at := login
		for {
			at = at.Add(24 * time.Hour)
			next, _, err := token.Rotate(at)
			if err != nil {
				require.ErrorIs(t, err, domain.ErrRefreshTokenExpired)
				require.False(t, at.Before(end), "venció antes de la vida máxima")
				break
			}
			require.False(t, next.ExpiresAt().After(end))
			token = next
		}
	})
}
