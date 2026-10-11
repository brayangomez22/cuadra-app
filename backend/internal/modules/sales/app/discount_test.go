package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDiscountLimit(t *testing.T) {
	t.Run("devuelve el tope de la empresa a quien no tiene sales:discount", func(t *testing.T) {
		e := newEnv(t)
		e.setMaxDiscount(t, "0.10")

		limit, err := e.svc.GetDiscountLimit(t.Context(), e.actor)
		require.NoError(t, err)
		require.False(t, limit.Unlimited())
		requireDec(t, "0.10", limit.Max())
		e.span(t, "sales.GetDiscountLimit")
	})

	t.Run("sin política configurada el tope es 0", func(t *testing.T) {
		e := newEnv(t)

		limit, err := e.svc.GetDiscountLimit(t.Context(), e.actor)
		require.NoError(t, err)
		require.False(t, limit.Unlimited())
		requireDec(t, "0", limit.Max())
	})

	t.Run("no tiene tope con sales:discount", func(t *testing.T) {
		e := newEnv(t)
		e.setMaxDiscount(t, "0.10")
		e.actor.UnlimitedDiscount = true

		limit, err := e.svc.GetDiscountLimit(t.Context(), e.actor)
		require.NoError(t, err)
		require.True(t, limit.Unlimited())
	})
}
