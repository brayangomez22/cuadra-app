package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/config"
)

func TestLoadDemo(t *testing.T) {
	base := map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": jwtSecret}
	with := func(extra map[string]string) map[string]string {
		vars := map[string]string{}
		for k, v := range base {
			vars[k] = v
		}
		for k, v := range extra {
			vars[k] = v
		}
		return vars
	}

	t.Run("la demo está apagada por defecto", func(t *testing.T) {
		cfg, err := config.Load(env(base))
		require.NoError(t, err)
		require.False(t, cfg.DemoEnabled)
	})

	t.Run("enciende la demo con su contraseña", func(t *testing.T) {
		cfg, err := config.Load(env(with(map[string]string{"DEMO_ENABLED": "true", "DEMO_PASSWORD": "demo-clave-segura"})))
		require.NoError(t, err)
		require.True(t, cfg.DemoEnabled)
		require.Equal(t, "demo-clave-segura", cfg.DemoPassword)
	})

	t.Run("la demo encendida exige una contraseña de 12 caracteres o más sin mostrarla", func(t *testing.T) {
		_, err := config.Load(env(with(map[string]string{"DEMO_ENABLED": "true", "DEMO_PASSWORD": "corta"})))
		require.ErrorContains(t, err, "DEMO_PASSWORD")
		require.NotContains(t, err.Error(), "corta")
	})

	t.Run("rechaza un DEMO_ENABLED que no es booleano", func(t *testing.T) {
		_, err := config.Load(env(with(map[string]string{"DEMO_ENABLED": "quizás"})))
		require.ErrorContains(t, err, "DEMO_ENABLED")
	})
}
