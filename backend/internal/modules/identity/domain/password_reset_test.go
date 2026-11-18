package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestPasswordReset(t *testing.T) {
	now := time.Date(2026, 11, 18, 15, 0, 0, 0, time.UTC)
	tenantID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	byID := uuid.Must(uuid.NewV7())

	t.Run("crea un restablecimiento vigente por un día", func(t *testing.T) {
		r, raw, err := domain.NewPasswordReset(tenantID, userID, byID, now)
		require.NoError(t, err)
		require.Equal(t, userID, r.UserID())
		require.Equal(t, byID, r.IssuedBy())
		require.Equal(t, now.Add(domain.PasswordResetTTL), r.ExpiresAt())
		gotTenant, gotHash, err := domain.ParseLinkToken(raw)
		require.NoError(t, err)
		require.Equal(t, tenantID, gotTenant)
		require.Equal(t, r.Hash(), gotHash)
	})

	t.Run("rechaza un restablecimiento sin usuario", func(t *testing.T) {
		_, _, err := domain.NewPasswordReset(tenantID, uuid.Nil, byID, now)
		require.ErrorIs(t, err, domain.ErrInvalidUserID)
	})

	t.Run("el restablecimiento de contraseña se usa una sola vez", func(t *testing.T) {
		r, _, err := domain.NewPasswordReset(tenantID, userID, byID, now)
		require.NoError(t, err)
		require.NoError(t, r.Use(now.Add(time.Minute)))
		require.ErrorIs(t, r.Use(now.Add(2*time.Minute)), domain.ErrLinkUsed)
	})

	t.Run("el restablecimiento vencido no se puede usar", func(t *testing.T) {
		r, _, err := domain.NewPasswordReset(tenantID, userID, byID, now)
		require.NoError(t, err)
		require.ErrorIs(t, r.Use(now.Add(domain.PasswordResetTTL)), domain.ErrLinkExpired)
	})

	t.Run("se rehidrata usado", func(t *testing.T) {
		r, _, err := domain.NewPasswordReset(tenantID, userID, byID, now)
		require.NoError(t, err)
		require.NoError(t, r.Use(now))
		got, err := domain.RehydratePasswordReset(r.Snapshot())
		require.NoError(t, err)
		require.ErrorIs(t, got.Use(now), domain.ErrLinkUsed)
	})
}
