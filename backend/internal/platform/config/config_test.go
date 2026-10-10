package config_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/config"
)

// jwtSecret is a valid JWT_SECRET (32 bytes) for tests.
const jwtSecret = "0123456789abcdef0123456789abcdef"

func env(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestLoad(t *testing.T) {
	t.Run("falla si falta DATABASE_URL", func(t *testing.T) {
		_, err := config.Load(env(map[string]string{}))

		require.ErrorContains(t, err, "DATABASE_URL")
	})

	t.Run("reporta todos los errores de configuración juntos", func(t *testing.T) {
		_, err := config.Load(env(map[string]string{
			"LOG_LEVEL":        "ruidoso",
			"SHUTDOWN_TIMEOUT": "pronto",
		}))

		require.ErrorContains(t, err, "DATABASE_URL")
		require.ErrorContains(t, err, "JWT_SECRET")
		require.ErrorContains(t, err, "LOG_LEVEL")
		require.ErrorContains(t, err, "SHUTDOWN_TIMEOUT")
	})

	t.Run("aplica valores por defecto a las opcionales", func(t *testing.T) {
		cfg, err := config.Load(env(map[string]string{
			"DATABASE_URL": "postgres://localhost/cuadra",
			"JWT_SECRET":   jwtSecret,
		}))

		require.NoError(t, err)
		require.Equal(t, config.Config{
			Env:             "development",
			HTTPAddr:        ":8080",
			DatabaseURL:     "postgres://localhost/cuadra",
			JWTSecret:       []byte(jwtSecret),
			LogLevel:        slog.LevelInfo,
			ShutdownTimeout: 15 * time.Second,
		}, cfg)
	})

	t.Run("lee los valores definidos", func(t *testing.T) {
		cfg, err := config.Load(env(map[string]string{
			"APP_ENV":          "production",
			"HTTP_ADDR":        ":9090",
			"DATABASE_URL":     "postgres://db/cuadra",
			"JWT_SECRET":       jwtSecret,
			"LOG_LEVEL":        "debug",
			"SHUTDOWN_TIMEOUT": "30s",
		}))

		require.NoError(t, err)
		require.Equal(t, "production", cfg.Env)
		require.Equal(t, ":9090", cfg.HTTPAddr)
		require.Equal(t, slog.LevelDebug, cfg.LogLevel)
		require.Equal(t, 30*time.Second, cfg.ShutdownTimeout)
	})

	t.Run("rechaza un SHUTDOWN_TIMEOUT inválido", func(t *testing.T) {
		for _, value := range []string{"pronto", "0s", "-5s"} {
			_, err := config.Load(env(map[string]string{
				"DATABASE_URL":     "postgres://localhost/cuadra",
				"JWT_SECRET":       jwtSecret,
				"SHUTDOWN_TIMEOUT": value,
			}))

			require.ErrorContains(t, err, "SHUTDOWN_TIMEOUT", "valor %q", value)
		}
	})

	t.Run("falla si falta JWT_SECRET", func(t *testing.T) {
		_, err := config.Load(env(map[string]string{"DATABASE_URL": "postgres://localhost/cuadra"}))

		require.ErrorContains(t, err, "JWT_SECRET is required")
	})

	t.Run("rechaza un JWT_SECRET de menos de 32 bytes sin mostrarlo", func(t *testing.T) {
		_, err := config.Load(env(map[string]string{
			"DATABASE_URL": "postgres://localhost/cuadra",
			"JWT_SECRET":   "secreto-corto",
		}))

		require.ErrorContains(t, err, "JWT_SECRET must have at least 32 bytes")
		require.NotContains(t, err.Error(), "secreto-corto")
	})
}
