package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/identity/domain"
)

func TestUpdateTenant(t *testing.T) {
	rate := func(s string) *decimal.Decimal {
		d := decimal.RequireFromString(s)
		return &d
	}

	t.Run("actualiza el IVA por defecto de la empresa del usuario", func(t *testing.T) {
		tests := []struct{ template, rate string }{
			{template: "ferreteria", rate: "0.05"},
			{template: "drogueria", rate: "0"},
		}
		for _, tt := range tests {
			e := newEnv(t)
			in := signUpInput()
			in.BusinessTemplate = tt.template
			session, err := e.svc.SignUp(t.Context(), in)
			require.NoError(t, err)
			tenantID, userID := session.Profile.Tenant.ID(), session.Profile.User.ID()

			tenant, err := e.svc.UpdateTenant(t.Context(), tenantID, userID, app.TenantUpdate{DefaultTaxRate: rate(tt.rate)})

			require.NoError(t, err, tt.template)
			require.Equal(t, tt.rate, tenant.DefaultTaxRate().String(), tt.template)
			stored, err := e.tenants.GetByID(t.Context(), tenantID)
			require.NoError(t, err)
			require.Equal(t, tt.rate, stored.DefaultTaxRate().String(), tt.template)
			span := e.span(t, "identity.UpdateTenant")
			require.Contains(t, span.Attributes(), attribute.String("tenant.id", tenantID.String()))
			require.Contains(t, span.Attributes(), attribute.String("user.id", userID.String()))
		}
	})

	t.Run("sin cambios devuelve la empresa como está", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "ana@eltornillo.co", validPassword, domain.RoleOwner)

		got, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), user.ID(), app.TenantUpdate{})

		require.NoError(t, err)
		require.Equal(t, tenant.DefaultTaxRate().String(), got.DefaultTaxRate().String())
	})

	t.Run("un IVA inválido no se guarda", func(t *testing.T) {
		e := newEnv(t)
		tenant, user := e.seed(t, "ana@eltornillo.co", validPassword, domain.RoleOwner)

		_, err := e.svc.UpdateTenant(t.Context(), tenant.ID(), user.ID(), app.TenantUpdate{DefaultTaxRate: rate("1")})

		require.ErrorIs(t, err, domain.ErrInvalidDefaultTaxRate)
		stored, err := e.tenants.GetByID(t.Context(), tenant.ID())
		require.NoError(t, err)
		require.Equal(t, tenant.DefaultTaxRate().String(), stored.DefaultTaxRate().String())
		require.Equal(t, codes.Error, e.span(t, "identity.UpdateTenant").Status().Code)
	})

	t.Run("una empresa que no existe responde ErrTenantNotFound", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.UpdateTenant(t.Context(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), app.TenantUpdate{DefaultTaxRate: rate("0.19")})

		require.ErrorIs(t, err, domain.ErrTenantNotFound)
	})
}
