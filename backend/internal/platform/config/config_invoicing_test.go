package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/config"
)

func TestLoadInvoicing(t *testing.T) {
	vars := func(extra map[string]string) func(string) string {
		v := map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": jwtSecret}
		for k, val := range extra {
			v[k] = val
		}
		return env(v)
	}

	t.Run("fuera de producción el proveedor por defecto es el simulado", func(t *testing.T) {
		cfg, err := config.Load(vars(nil))
		require.NoError(t, err)
		require.Equal(t, "simulated", cfg.InvoicingProvider)
	})

	t.Run("en producción exige elegir el proveedor", func(t *testing.T) {
		_, err := config.Load(vars(map[string]string{"APP_ENV": "production"}))
		require.ErrorContains(t, err, "INVOICING_PROVIDER")
	})

	t.Run("en producción rechaza el proveedor simulado: los documentos no tendrían validez", func(t *testing.T) {
		_, err := config.Load(vars(map[string]string{"APP_ENV": "production", "INVOICING_PROVIDER": "simulated"}))
		require.ErrorContains(t, err, "INVOICING_PROVIDER")
	})

	t.Run("rechaza un proveedor desconocido", func(t *testing.T) {
		_, err := config.Load(vars(map[string]string{"INVOICING_PROVIDER": "otro"}))
		require.ErrorContains(t, err, "INVOICING_PROVIDER")
	})
}
