package httpadapter_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
)

// settingsStub answers the stock settings use cases.
type settingsStub struct {
	policy   domain.StockPolicy
	update   app.StockSettingsUpdate
	settings app.Actor
}

func (s *settingsStub) StockSettings(_ context.Context, a app.Actor) (domain.StockPolicy, error) {
	s.settings = a
	return s.policy, nil
}

func (s *settingsStub) UpdateStockSettings(_ context.Context, a app.Actor, in app.StockSettingsUpdate) (domain.StockPolicy, error) {
	s.settings, s.update = a, in
	if in.AllowNegative != nil {
		s.policy.AllowNegative = *in.AllowNegative
	}
	return s.policy, nil
}

func TestInventorySettings(t *testing.T) {
	t.Run("devuelve las reglas de inventario de la empresa del token", func(t *testing.T) {
		uc := &stubUseCases{settingsStub: settingsStub{policy: domain.StockPolicy{AllowNegative: true}}}

		rec := serve(t, uc, &owner, http.MethodGet, "/api/v1/inventory/settings", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"allow_negative_stock": true}`, rec.Body.String())
		require.Equal(t, owner.TenantID, uc.settings.TenantID)
	})

	t.Run("pasa el cambio al caso de uso y responde las reglas nuevas", func(t *testing.T) {
		uc := &stubUseCases{}

		rec := serve(t, uc, &owner, http.MethodPatch, "/api/v1/inventory/settings", `{"allow_negative_stock": true}`)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.JSONEq(t, `{"allow_negative_stock": true}`, rec.Body.String())
		require.True(t, *uc.update.AllowNegative)
		require.Equal(t, owner.UserID, uc.settings.UserID)
	})

	t.Run("sin usuario autenticado responde 401", func(t *testing.T) {
		requireError(t, serve(t, &stubUseCases{}, nil, http.MethodGet, "/api/v1/inventory/settings", ""), http.StatusUnauthorized, "unauthorized")
	})
}
