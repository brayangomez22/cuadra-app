package config_test

import (
	"encoding/base64"
	"net/netip"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/config"
)

func base(extra map[string]string) map[string]string {
	vars := map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": jwtSecret}
	for k, v := range extra {
		vars[k] = v
	}
	return vars
}

func TestLoadSecurity(t *testing.T) {
	t.Run("por defecto no confía en proxies, no permite otros orígenes ni llaves anteriores", func(t *testing.T) {
		cfg, err := config.Load(env(base(nil)))
		require.NoError(t, err)
		require.Empty(t, cfg.TrustedProxies)
		require.Empty(t, cfg.CORSAllowedOrigins)
		require.Nil(t, cfg.JWTPreviousSecret)
		require.Nil(t, cfg.InvoicingPreviousTokenKey)
		require.False(t, cfg.HSTS())
	})

	t.Run("lee los proxies de confianza (rangos o IPs) y los orígenes permitidos", func(t *testing.T) {
		cfg, err := config.Load(env(base(map[string]string{
			"TRUSTED_PROXIES":      "10.0.0.0/16, 100.64.0.1",
			"CORS_ALLOWED_ORIGINS": "https://app.cuadra.co, https://demo.cuadra.co",
		})))
		require.NoError(t, err)
		require.Equal(t, []netip.Prefix{netip.MustParsePrefix("10.0.0.0/16"), netip.MustParsePrefix("100.64.0.1/32")}, cfg.TrustedProxies)
		require.Equal(t, []string{"https://app.cuadra.co", "https://demo.cuadra.co"}, cfg.CORSAllowedOrigins)
	})

	t.Run("rechaza un proxy o un origen inválido", func(t *testing.T) {
		_, err := config.Load(env(base(map[string]string{"TRUSTED_PROXIES": "la-red"})))
		require.ErrorContains(t, err, "TRUSTED_PROXIES")
		_, err = config.Load(env(base(map[string]string{"CORS_ALLOWED_ORIGINS": "app.cuadra.co/"})))
		require.ErrorContains(t, err, "CORS_ALLOWED_ORIGINS")
		_, err = config.Load(env(base(map[string]string{"CORS_ALLOWED_ORIGINS": "*"})))
		require.ErrorContains(t, err, "CORS_ALLOWED_ORIGINS")
	})

	t.Run("acepta la llave anterior del JWT para rotarla, del mismo largo mínimo y distinta", func(t *testing.T) {
		previous := strings.Repeat("p", 32)
		cfg, err := config.Load(env(base(map[string]string{"JWT_SECRET_PREVIOUS": previous})))
		require.NoError(t, err)
		require.Equal(t, []byte(previous), cfg.JWTPreviousSecret)

		_, err = config.Load(env(base(map[string]string{"JWT_SECRET_PREVIOUS": "corta"})))
		require.ErrorContains(t, err, "JWT_SECRET_PREVIOUS")
		_, err = config.Load(env(base(map[string]string{"JWT_SECRET_PREVIOUS": jwtSecret})))
		require.ErrorContains(t, err, "JWT_SECRET_PREVIOUS")
	})

	t.Run("acepta la llave anterior de los tokens del proveedor solo con una llave actual", func(t *testing.T) {
		key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
		previous := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("o", 32)))
		cfg, err := config.Load(env(base(map[string]string{"INVOICING_TOKEN_KEY": key, "INVOICING_TOKEN_KEY_PREVIOUS": previous})))
		require.NoError(t, err)
		require.Equal(t, []byte(strings.Repeat("o", 32)), cfg.InvoicingPreviousTokenKey)

		_, err = config.Load(env(base(map[string]string{"INVOICING_TOKEN_KEY_PREVIOUS": previous})))
		require.ErrorContains(t, err, "INVOICING_TOKEN_KEY_PREVIOUS")
		_, err = config.Load(env(base(map[string]string{"INVOICING_TOKEN_KEY": key, "INVOICING_TOKEN_KEY_PREVIOUS": "corta"})))
		require.ErrorContains(t, err, "INVOICING_TOKEN_KEY_PREVIOUS")
	})

	t.Run("los límites de solicitudes están activos salvo que se apaguen fuera de producción", func(t *testing.T) {
		cfg, err := config.Load(env(base(nil)))
		require.NoError(t, err)
		require.True(t, cfg.RateLimitsEnabled)

		cfg, err = config.Load(env(base(map[string]string{"RATE_LIMITS_ENABLED": "false"})))
		require.NoError(t, err)
		require.False(t, cfg.RateLimitsEnabled)

		_, err = config.Load(env(base(map[string]string{"RATE_LIMITS_ENABLED": "quizás"})))
		require.ErrorContains(t, err, "RATE_LIMITS_ENABLED")
		_, err = config.Load(env(base(map[string]string{"APP_ENV": "production", "RATE_LIMITS_ENABLED": "false"})))
		require.ErrorContains(t, err, "RATE_LIMITS_ENABLED")
	})

	t.Run("HSTS solo en producción", func(t *testing.T) {
		require.True(t, config.Config{Env: "production"}.HSTS())
		require.False(t, config.Config{Env: "development"}.HSTS())
	})
}
