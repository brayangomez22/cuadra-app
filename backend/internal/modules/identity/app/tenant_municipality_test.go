package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestUpdateTenantMunicipality(t *testing.T) {
	code := func(s string) *string { return &s }

	t.Run("guarda el municipio DANE del emisor y conserva lo demás", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "ana@eltornillo.co", validPassword, domain.RoleOwner)
		_, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), user.ID(), app.TenantUpdate{City: code("Medellín")})
		require.NoError(t, err)

		got, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), user.ID(), app.TenantUpdate{MunicipalityCode: code("05001")})

		require.NoError(t, err)
		require.Equal(t, "05001", got.Contact().MunicipalityCode())
		require.Equal(t, "Medellín", got.Contact().City())
		stored, err := e.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.Equal(t, "05001", stored.Contact().MunicipalityCode())
	})

	t.Run("cambiar la dirección no borra el municipio", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "ana@eltornillo.co", validPassword, domain.RoleOwner)
		_, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), user.ID(), app.TenantUpdate{MunicipalityCode: code("11001")})
		require.NoError(t, err)

		got, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), user.ID(), app.TenantUpdate{Address: code("Cl 72 # 10-15")})

		require.NoError(t, err)
		require.Equal(t, "11001", got.Contact().MunicipalityCode())
	})

	t.Run("un código mal formado no cambia nada", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "ana@eltornillo.co", validPassword, domain.RoleOwner)

		_, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), user.ID(), app.TenantUpdate{MunicipalityCode: code("5001")})

		require.ErrorIs(t, err, domain.ErrInvalidMunicipality)
		stored, err := e.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.Empty(t, stored.Contact().MunicipalityCode())
	})
}

func TestTenantProfile(t *testing.T) {
	t.Run("lee la empresa del emisor de los documentos con su municipio", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "ana@eltornillo.co", validPassword, domain.RoleOwner)
		_, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), user.ID(), app.TenantUpdate{MunicipalityCode: new("05001")})
		require.NoError(t, err)

		got, err := e.svc.Tenant(t.Context(), tenant.ID())

		require.NoError(t, err)
		require.Equal(t, tenant.NIT(), got.NIT())
		require.Equal(t, "05001", got.Contact().MunicipalityCode())
		require.False(t, got.IsDemo())
		require.Contains(t, e.span(t, "identity.GetTenant").Attributes(), attribute.String("tenant.id", tenant.ID().String()))
	})
}
