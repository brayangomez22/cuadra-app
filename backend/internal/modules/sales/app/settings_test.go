package app_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestSalesSettings(t *testing.T) {
	t.Run("una empresa sin reglas no deja descontar y exige caja", func(t *testing.T) {
		e := newEnv(t)
		delete(e.store.policies, e.actor.TenantID)

		got, err := e.svc.SalesSettings(t.Context(), e.actor)

		require.NoError(t, err)
		require.True(t, got.MaxDiscount.Max().IsZero())
		require.False(t, got.CashSessionOptional)
		_, attrs := e.span(t, "sales.GetSalesSettings")
		require.Equal(t, e.actor.TenantID.String(), attrs["tenant.id"].AsString())
	})
}

func TestUpdateSalesSettings(t *testing.T) {
	fraction := func(s string) *decimal.Decimal {
		d := decimal.RequireFromString(s)
		return &d
	}
	yes, no := true, false

	t.Run("cambia el descuento máximo y la caja obligatoria de negocios distintos", func(t *testing.T) {
		tests := []struct {
			name        string
			maxDiscount string
			requireCash bool
		}{
			// A pharmacy that never discounts at the counter, and a clothing
			// store whose cashiers discount up to 15 % at the end of season.
			{name: "droguería", maxDiscount: "0", requireCash: true},
			{name: "almacén de ropa", maxDiscount: "0.15", requireCash: false},
		}
		for _, tt := range tests {
			e := newEnv(t)

			got, err := e.svc.UpdateSalesSettings(t.Context(), e.actor, app.SalesSettingsUpdate{
				MaxDiscount: fraction(tt.maxDiscount), RequireCashSession: &tt.requireCash,
			})

			require.NoError(t, err, tt.name)
			require.Equal(t, tt.maxDiscount, got.MaxDiscount.Max().String(), tt.name)
			require.Equal(t, !tt.requireCash, got.CashSessionOptional, tt.name)
			require.Equal(t, got, e.store.policies[e.actor.TenantID], tt.name)
			_, attrs := e.span(t, "sales.UpdateSalesSettings")
			require.Equal(t, e.actor.UserID.String(), attrs["user.id"].AsString())
			require.Equal(t, int64(2), e.counter(t, "cuadra.sales.settings_changes"), tt.name)
		}
	})

	t.Run("conserva lo que no se envía", func(t *testing.T) {
		e := newEnv(t)
		limit, err := domain.NewDiscountLimit(decimal.RequireFromString("0.1"))
		require.NoError(t, err)
		e.store.policies[e.actor.TenantID] = domain.SalesPolicy{MaxDiscount: limit}

		got, err := e.svc.UpdateSalesSettings(t.Context(), e.actor, app.SalesSettingsUpdate{RequireCashSession: &no})

		require.NoError(t, err)
		require.Equal(t, "0.1", got.MaxDiscount.Max().String())
		require.True(t, got.CashSessionOptional)
		require.Equal(t, int64(1), e.counter(t, "cuadra.sales.settings_changes"))
	})

	t.Run("un descuento fuera de 0 a 1 no guarda ningún cambio", func(t *testing.T) {
		for _, bad := range []string{"-0.01", "1.5", "0.12345"} {
			e := newEnv(t)
			before := e.store.policies[e.actor.TenantID]

			_, err := e.svc.UpdateSalesSettings(t.Context(), e.actor, app.SalesSettingsUpdate{
				MaxDiscount: fraction(bad), RequireCashSession: &yes,
			})

			require.ErrorIs(t, err, domain.ErrInvalidDiscountLimit, bad)
			require.Equal(t, before, e.store.policies[e.actor.TenantID], bad)
			span, _ := e.span(t, "sales.UpdateSalesSettings")
			require.Equal(t, codes.Error, span.Status().Code)
		}
	})

	t.Run("sin cambios no guarda nada", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.UpdateSalesSettings(t.Context(), e.actor, app.SalesSettingsUpdate{})

		require.NoError(t, err)
		require.Zero(t, e.counter(t, "cuadra.sales.settings_changes"))
	})
}
