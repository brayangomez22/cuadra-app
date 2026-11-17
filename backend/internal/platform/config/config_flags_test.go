package config_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/config"
)

func TestFlagsConfig(t *testing.T) {
	t.Run("sin operadores ni TTL, nadie maneja las flags y la caché dura 30 segundos", func(t *testing.T) {
		cfg, err := config.Load(env(base(nil)))
		require.NoError(t, err)
		require.Empty(t, cfg.PlatformOperators)
		require.Equal(t, 30*time.Second, cfg.FlagsCacheTTL)
	})

	t.Run("lee los operadores y el TTL", func(t *testing.T) {
		a, b := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
		cfg, err := config.Load(env(base(map[string]string{
			"PLATFORM_OPERATORS":      a.String() + ", " + b.String(),
			"FEATURE_FLAGS_CACHE_TTL": "5s",
		})))
		require.NoError(t, err)
		require.Equal(t, []uuid.UUID{a, b}, cfg.PlatformOperators)
		require.Equal(t, 5*time.Second, cfg.FlagsCacheTTL)
	})

	t.Run("rechaza un operador que no es un ID y un TTL que no es positivo", func(t *testing.T) {
		_, err := config.Load(env(base(map[string]string{"PLATFORM_OPERATORS": "brayan"})))
		require.ErrorContains(t, err, "PLATFORM_OPERATORS")
		_, err = config.Load(env(base(map[string]string{"FEATURE_FLAGS_CACHE_TTL": "0s"})))
		require.ErrorContains(t, err, "FEATURE_FLAGS_CACHE_TTL")
	})
}
