package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func mustNIT(t *testing.T, s string) domain.NIT {
	t.Helper()
	nit, err := domain.ParseNIT(s)
	require.NoError(t, err)
	return nit
}

func TestNewTenant(t *testing.T) {
	now := time.Date(2026, 10, 9, 15, 0, 0, 0, time.FixedZone("COT", -5*3600))
	vat := decimal.RequireFromString("0.19")

	t.Run("crea tenant activo con id v7", func(t *testing.T) {
		tenant, err := domain.NewTenant("  Ferretería El Tornillo  ", mustNIT(t, "890903938-8"), "ferreteria", vat, now)
		require.NoError(t, err)
		require.Equal(t, 7, int(tenant.ID().Version()))
		require.Equal(t, "Ferretería El Tornillo", tenant.Name())
		require.Equal(t, "890903938-8", tenant.NIT().String())
		require.Equal(t, "ferreteria", tenant.BusinessTemplate())
		require.Equal(t, domain.TenantActive, tenant.Status())
		require.True(t, tenant.IsActive())
		require.Equal(t, now.UTC(), tenant.CreatedAt())
		require.Equal(t, time.UTC, tenant.CreatedAt().Location())
	})

	t.Run("rechaza tenant sin nombre o con nombre demasiado largo", func(t *testing.T) {
		for _, name := range []string{"", "   ", strings.Repeat("a", 201)} {
			_, err := domain.NewTenant(name, mustNIT(t, "890903938-8"), "ferreteria", vat, now)
			require.ErrorIs(t, err, domain.ErrInvalidTenantName)
		}
	})

	t.Run("rechaza tenant sin NIT", func(t *testing.T) {
		_, err := domain.NewTenant("Ferretería El Tornillo", domain.NIT{}, "ferreteria", vat, now)
		require.ErrorIs(t, err, domain.ErrInvalidNIT)
	})

	t.Run("guarda la plantilla de su tipo de negocio", func(t *testing.T) {
		for _, template := range []string{"ferreteria", "ropa_calzado", "generica"} {
			tenant, err := domain.NewTenant("Mi negocio", mustNIT(t, "890903938-8"), template, vat, now)
			require.NoError(t, err)
			require.Equal(t, template, tenant.BusinessTemplate())
		}
	})

	t.Run("rechaza una plantilla vacía o mal escrita", func(t *testing.T) {
		for _, template := range []string{"", " ", "Ferreteria", "ropa-calzado", "1tienda", "../generica", strings.Repeat("a", 65)} {
			_, err := domain.NewTenant("Mi negocio", mustNIT(t, "890903938-8"), template, vat, now)
			require.ErrorIs(t, err, domain.ErrUnknownBusinessTemplate, template)
		}
	})

	t.Run("guarda el IVA por defecto de su plantilla", func(t *testing.T) {
		tests := []struct {
			template string
			rate     string
		}{
			{template: "ferreteria", rate: "0.19"},
			{template: "drogueria", rate: "0"},
			{template: "ropa_calzado", rate: "0.05"},
		}
		for _, tt := range tests {
			tenant, err := domain.NewTenant("Mi negocio", mustNIT(t, "890903938-8"), tt.template, decimal.RequireFromString(tt.rate), now)
			require.NoError(t, err)
			require.Equal(t, tt.rate, tenant.DefaultTaxRate().String(), tt.template)
		}
	})

	t.Run("rechaza un IVA por defecto negativo, de 100 % o más, o con más de 4 decimales", func(t *testing.T) {
		for _, rate := range []string{"-0.01", "1", "1.19", "0.12345"} {
			_, err := domain.NewTenant("Mi negocio", mustNIT(t, "890903938-8"), "generica", decimal.RequireFromString(rate), now)
			require.ErrorIs(t, err, domain.ErrInvalidDefaultTaxRate, rate)
		}
	})

	t.Run("suspende y reactiva un tenant", func(t *testing.T) {
		tenant, err := domain.NewTenant("Ferretería El Tornillo", mustNIT(t, "890903938-8"), "ferreteria", vat, now)
		require.NoError(t, err)

		tenant.Suspend()
		require.Equal(t, domain.TenantSuspended, tenant.Status())
		require.False(t, tenant.IsActive())

		tenant.Activate()
		require.True(t, tenant.IsActive())
	})
}

func TestTenantSetDefaultTaxRate(t *testing.T) {
	now := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)

	t.Run("cambia el IVA por defecto de la empresa", func(t *testing.T) {
		tests := []struct {
			template string
			from, to string
		}{
			{template: "ferreteria", from: "0.19", to: "0.05"},
			{template: "drogueria", from: "0", to: "0.19"},
			{template: "ropa_calzado", from: "0.19", to: "0"},
		}
		for _, tt := range tests {
			tenant, err := domain.NewTenant("Mi negocio", mustNIT(t, "890903938-8"), tt.template, decimal.RequireFromString(tt.from), now)
			require.NoError(t, err)
			require.NoError(t, tenant.SetDefaultTaxRate(decimal.RequireFromString(tt.to)), tt.template)
			require.Equal(t, tt.to, tenant.DefaultTaxRate().String(), tt.template)
		}
	})

	t.Run("rechaza un IVA negativo, de 100 % o más, o con más de 4 decimales sin cambiar el actual", func(t *testing.T) {
		tenant, err := domain.NewTenant("Mi negocio", mustNIT(t, "890903938-8"), "papeleria", decimal.RequireFromString("0.19"), now)
		require.NoError(t, err)
		for _, rate := range []string{"-0.01", "1", "1.19", "0.12345"} {
			require.ErrorIs(t, tenant.SetDefaultTaxRate(decimal.RequireFromString(rate)), domain.ErrInvalidDefaultTaxRate, rate)
			require.Equal(t, "0.19", tenant.DefaultTaxRate().String(), rate)
		}
	})
}
