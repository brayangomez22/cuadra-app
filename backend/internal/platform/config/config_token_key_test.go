package config_test

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/config"
)

func TestLoadInvoicingTokenKey(t *testing.T) {
	vars := func(extra map[string]string) func(string) string {
		v := map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": jwtSecret}
		for k, val := range extra {
			v[k] = val
		}
		return env(v)
	}
	aKey := bytes.Repeat([]byte{7}, 32)

	t.Run("lee la llave de los tokens del proveedor en base64", func(t *testing.T) {
		cfg, err := config.Load(vars(map[string]string{"INVOICING_TOKEN_KEY": base64.StdEncoding.EncodeToString(aKey)}))
		require.NoError(t, err)
		require.Equal(t, aKey, cfg.InvoicingTokenKey)
	})

	t.Run("fuera de producción es opcional", func(t *testing.T) {
		cfg, err := config.Load(vars(nil))
		require.NoError(t, err)
		require.Empty(t, cfg.InvoicingTokenKey)
	})

	t.Run("en producción es obligatoria", func(t *testing.T) {
		_, err := config.Load(vars(map[string]string{"APP_ENV": "production"}))
		require.ErrorContains(t, err, "INVOICING_TOKEN_KEY")
	})

	t.Run("rechaza una llave que no es base64 o no tiene 32 bytes", func(t *testing.T) {
		for _, raw := range []string{"no es base64!", base64.StdEncoding.EncodeToString(aKey[:16])} {
			_, err := config.Load(vars(map[string]string{"INVOICING_TOKEN_KEY": raw}))
			require.ErrorContains(t, err, "INVOICING_TOKEN_KEY", raw)
		}
	})

	t.Run("el error no muestra la llave", func(t *testing.T) {
		raw := base64.StdEncoding.EncodeToString(aKey[:16])
		_, err := config.Load(vars(map[string]string{"INVOICING_TOKEN_KEY": raw}))
		require.NotContains(t, err.Error(), raw)
	})
}
