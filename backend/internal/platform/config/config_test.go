package config_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/config"
)

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
		require.ErrorContains(t, err, "LOG_LEVEL")
		require.ErrorContains(t, err, "SHUTDOWN_TIMEOUT")
	})

	t.Run("aplica valores por defecto a las opcionales", func(t *testing.T) {
		cfg, err := config.Load(env(map[string]string{
			"DATABASE_URL": "postgres://localhost/cuadra",
		}))

		require.NoError(t, err)
		require.Equal(t, config.Config{
			Env:             "development",
			HTTPAddr:        ":8080",
			DatabaseURL:     "postgres://localhost/cuadra",
			LogLevel:        slog.LevelInfo,
			ShutdownTimeout: 15 * time.Second,
		}, cfg)
	})

	t.Run("lee los valores definidos", func(t *testing.T) {
		cfg, err := config.Load(env(map[string]string{
			"APP_ENV":          "production",
			"HTTP_ADDR":        ":9090",
			"DATABASE_URL":     "postgres://db/cuadra",
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
				"SHUTDOWN_TIMEOUT": value,
			}))

			require.ErrorContains(t, err, "SHUTDOWN_TIMEOUT", "valor %q", value)
		}
	})
}
