package config_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/config"
)

func TestAnalyticsConfig(t *testing.T) {
	t.Run("sin clave ni webhook la analítica está apagada y los comentarios van al log", func(t *testing.T) {
		cfg, err := config.Load(env(base(nil)))
		require.NoError(t, err)
		require.Empty(t, cfg.AnalyticsSecret)
		require.Empty(t, cfg.FeedbackWebhookURL)
	})

	t.Run("lee la clave y el webhook", func(t *testing.T) {
		secret := strings.Repeat("k", 32)
		cfg, err := config.Load(env(base(map[string]string{
			"ANALYTICS_SECRET": secret, "FEEDBACK_WEBHOOK_URL": "https://chat.example.com/hooks/abc",
		})))
		require.NoError(t, err)
		require.Equal(t, []byte(secret), cfg.AnalyticsSecret)
		require.Equal(t, "https://chat.example.com/hooks/abc", cfg.FeedbackWebhookURL)
	})

	t.Run("rechaza una clave corta sin mostrarla y un webhook que no es una URL", func(t *testing.T) {
		_, err := config.Load(env(base(map[string]string{"ANALYTICS_SECRET": "corta-y-secreta"})))
		require.ErrorContains(t, err, "ANALYTICS_SECRET")
		require.NotContains(t, err.Error(), "corta-y-secreta")
		_, err = config.Load(env(base(map[string]string{"FEEDBACK_WEBHOOK_URL": "chat"})))
		require.ErrorContains(t, err, "FEEDBACK_WEBHOOK_URL")
	})

	t.Run("en producción el webhook usa https", func(t *testing.T) {
		_, err := config.Load(env(base(map[string]string{"APP_ENV": "production", "FEEDBACK_WEBHOOK_URL": "http://chat.example.com/h"})))
		require.ErrorContains(t, err, "https")
	})
}
