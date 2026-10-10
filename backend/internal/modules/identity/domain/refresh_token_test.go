package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestRefreshToken(t *testing.T) {
	now := time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)
	tenantID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())

	t.Run("crea un token vigente cuyo texto lleva el tenant y su hash", func(t *testing.T) {
		token, raw, err := domain.NewRefreshToken(tenantID, userID, now)
		require.NoError(t, err)
		require.Equal(t, 7, int(token.ID().Version()))
		require.Equal(t, tenantID, token.TenantID())
		require.Equal(t, userID, token.UserID())
		require.NotEqual(t, uuid.Nil, token.FamilyID())
		require.Equal(t, now.Add(domain.RefreshTokenTTL), token.ExpiresAt())
		require.False(t, token.IsRevoked())
		require.False(t, token.IsExpired(now))

		gotTenant, gotHash, err := domain.ParseRefreshToken(raw)
		require.NoError(t, err)
		require.Equal(t, tenantID, gotTenant)
		require.Equal(t, token.Hash(), gotHash)
		require.NotContains(t, string(token.Hash()), raw, "the stored hash must not be the token")
	})

	t.Run("dos tokens nuevos tienen secretos y familias distintas", func(t *testing.T) {
		a, rawA, err := domain.NewRefreshToken(tenantID, userID, now)
		require.NoError(t, err)
		b, rawB, err := domain.NewRefreshToken(tenantID, userID, now)
		require.NoError(t, err)
		require.NotEqual(t, rawA, rawB)
		require.NotEqual(t, a.Hash(), b.Hash())
		require.NotEqual(t, a.FamilyID(), b.FamilyID())
	})

	t.Run("el hash de un mismo secreto es estable", func(t *testing.T) {
		_, raw, err := domain.NewRefreshToken(tenantID, userID, now)
		require.NoError(t, err)
		_, h1, err := domain.ParseRefreshToken(raw)
		require.NoError(t, err)
		_, h2, err := domain.ParseRefreshToken(raw)
		require.NoError(t, err)
		require.Equal(t, h1, h2)
	})

	t.Run("el token rotado queda revocado y el sucesor hereda la familia", func(t *testing.T) {
		token, raw, err := domain.NewRefreshToken(tenantID, userID, now)
		require.NoError(t, err)
		later := now.Add(time.Hour)

		next, nextRaw, err := token.Rotate(later)
		require.NoError(t, err)

		require.True(t, token.IsRevoked())
		require.Equal(t, later, *token.RevokedAt())
		require.False(t, next.IsRevoked())
		require.Equal(t, token.FamilyID(), next.FamilyID())
		require.Equal(t, token.TenantID(), next.TenantID())
		require.Equal(t, token.UserID(), next.UserID())
		require.NotEqual(t, token.ID(), next.ID())
		require.NotEqual(t, raw, nextRaw)
		require.Equal(t, later.Add(domain.RefreshTokenTTL), next.ExpiresAt())
	})

	t.Run("no se puede rotar un token ya revocado", func(t *testing.T) {
		token, _, err := domain.NewRefreshToken(tenantID, userID, now)
		require.NoError(t, err)
		_, _, err = token.Rotate(now)
		require.NoError(t, err)

		_, _, err = token.Rotate(now)
		require.ErrorIs(t, err, domain.ErrRefreshTokenReused)
	})

	t.Run("un token vencido se reporta como vencido y no se puede rotar", func(t *testing.T) {
		token, _, err := domain.NewRefreshToken(tenantID, userID, now)
		require.NoError(t, err)
		expiry := now.Add(domain.RefreshTokenTTL)

		require.False(t, token.IsExpired(expiry.Add(-time.Second)))
		require.True(t, token.IsExpired(expiry))

		_, _, err = token.Rotate(expiry)
		require.ErrorIs(t, err, domain.ErrRefreshTokenExpired)
		require.False(t, token.IsRevoked(), "a failed rotation leaves the token as it was")
	})

	t.Run("rechaza tokens con formato inválido", func(t *testing.T) {
		_, valid, err := domain.NewRefreshToken(tenantID, userID, now)
		require.NoError(t, err)
		_, secret, _ := strings.Cut(valid, ".")

		tests := []struct {
			name string
			raw  string
		}{
			{name: "vacío", raw: ""},
			{name: "sin separador", raw: tenantID.String() + secret},
			{name: "tenant que no es uuid", raw: "ferreteria." + secret},
			{name: "tenant nulo", raw: uuid.Nil.String() + "." + secret},
			{name: "secreto que no es base64url", raw: tenantID.String() + ".***"},
			{name: "secreto corto", raw: tenantID.String() + ".YWJj"},
			{name: "secreto con relleno", raw: valid + "="},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, _, err := domain.ParseRefreshToken(tt.raw)
				require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
			})
		}
	})
}

func TestRehydrateRefreshToken(t *testing.T) {
	now := time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)
	valid := func() domain.RefreshTokenSnapshot {
		return domain.RefreshTokenSnapshot{
			ID:        uuid.Must(uuid.NewV7()),
			TenantID:  uuid.Must(uuid.NewV7()),
			UserID:    uuid.Must(uuid.NewV7()),
			FamilyID:  uuid.Must(uuid.NewV7()),
			Hash:      make([]byte, 32),
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: now,
		}
	}

	t.Run("reconstruye un token revocado", func(t *testing.T) {
		s := valid()
		s.RevokedAt = &now
		token, err := domain.RehydrateRefreshToken(s)
		require.NoError(t, err)
		require.True(t, token.IsRevoked())
		require.Equal(t, s.FamilyID, token.FamilyID())
	})

	t.Run("rechaza datos corruptos", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(*domain.RefreshTokenSnapshot)
		}{
			{name: "sin id", mutate: func(s *domain.RefreshTokenSnapshot) { s.ID = uuid.Nil }},
			{name: "sin tenant", mutate: func(s *domain.RefreshTokenSnapshot) { s.TenantID = uuid.Nil }},
			{name: "sin usuario", mutate: func(s *domain.RefreshTokenSnapshot) { s.UserID = uuid.Nil }},
			{name: "sin familia", mutate: func(s *domain.RefreshTokenSnapshot) { s.FamilyID = uuid.Nil }},
			{name: "hash de otro tamaño", mutate: func(s *domain.RefreshTokenSnapshot) { s.Hash = []byte("abc") }},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := valid()
				tt.mutate(&s)
				_, err := domain.RehydrateRefreshToken(s)
				require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
			})
		}
	})
}
