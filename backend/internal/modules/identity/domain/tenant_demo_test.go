package domain_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestTenantDemo(t *testing.T) {
	now := time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)
	vat := decimal.RequireFromString("0.19")
	cases := []struct{ name, template string }{
		{"Ferretería Demo", "ferreteria"},
		{"Almacén de Ropa Demo", "ropa_calzado"},
	}

	for _, c := range cases {
		t.Run("un tenant nuevo no es de demostración: "+c.template, func(t *testing.T) {
			tenant, err := domain.NewTenant(c.name, mustNIT(t, "900123456-8"), c.template, vat, now)
			require.NoError(t, err)
			require.False(t, tenant.IsDemo())
		})

		t.Run("se marca de demostración y lo conserva al leerse: "+c.template, func(t *testing.T) {
			tenant, err := domain.NewTenant(c.name, mustNIT(t, "900123456-8"), c.template, vat, now)
			require.NoError(t, err)
			tenant.MarkDemo()
			require.True(t, tenant.IsDemo())

			stored, err := domain.RehydrateTenant(domain.TenantSnapshot{
				ID: tenant.ID(), Name: tenant.Name(), NIT: tenant.NIT(), Status: tenant.Status(),
				BusinessTemplate: tenant.BusinessTemplate(), DefaultTaxRate: tenant.DefaultTaxRate(),
				Demo: true, CreatedAt: tenant.CreatedAt(),
			})
			require.NoError(t, err)
			require.True(t, stored.IsDemo())
		})
	}
}
