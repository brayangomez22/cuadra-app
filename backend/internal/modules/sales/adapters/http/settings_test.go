package httpadapter_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

// settingsStub answers the sales settings use cases.
type settingsStub struct {
	policy      domain.SalesPolicy
	update      app.SalesSettingsUpdate
	updateErr   error
	settingsFor app.Actor
}

func (s *settingsStub) SalesSettings(_ context.Context, a app.Actor) (domain.SalesPolicy, error) {
	s.settingsFor = a
	return s.policy, nil
}

func (s *settingsStub) UpdateSalesSettings(_ context.Context, a app.Actor, in app.SalesSettingsUpdate) (domain.SalesPolicy, error) {
	s.settingsFor, s.update = a, in
	return s.policy, s.updateErr
}

func TestSalesSettings(t *testing.T) {
	limit, err := domain.NewDiscountLimit(decimal.RequireFromString("0.15"))
	require.NoError(t, err)

	t.Run("devuelve las reglas de venta de la empresa del token", func(t *testing.T) {
		uc := &stubUseCases{settingsStub: settingsStub{policy: domain.SalesPolicy{MaxDiscount: limit, CashSessionOptional: true}}}

		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/sales/settings", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"max_discount": "0.15", "require_cash_session": false}`, rec.Body.String())
		require.Equal(t, owner.TenantID, uc.settingsFor.TenantID)
	})

	t.Run("una empresa sin reglas responde 0 y caja obligatoria", func(t *testing.T) {
		rec := serve(t, &stubUseCases{}, &owner, http.MethodGet, "/api/v1/sales/settings", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"max_discount": "0", "require_cash_session": true}`, rec.Body.String())
	})

	t.Run("pasa el descuento y la caja al caso de uso", func(t *testing.T) {
		uc := &stubUseCases{settingsStub: settingsStub{policy: domain.SalesPolicy{MaxDiscount: limit}}}

		rec := serve(t, uc, &owner, http.MethodPatch, "/api/v1/sales/settings", `{"max_discount": "0.15", "require_cash_session": true}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, "0.15", uc.update.MaxDiscount.String())
		require.True(t, *uc.update.RequireCashSession)
		require.Equal(t, owner.UserID, uc.settingsFor.UserID)
	})

	t.Run("un descuento fuera de rango responde 422 invalid_discount_limit", func(t *testing.T) {
		uc := &stubUseCases{settingsStub: settingsStub{updateErr: domain.ErrInvalidDiscountLimit}}

		rec := serve(t, uc, &owner, http.MethodPatch, "/api/v1/sales/settings", `{"max_discount": "1.5"}`)

		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), `"invalid_discount_limit"`)
	})
}
