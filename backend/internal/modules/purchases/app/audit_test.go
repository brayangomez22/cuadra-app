package app_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func TestAuditOfPurchaseVoids(t *testing.T) {
	t.Run("anular una compra deja un evento con su total", func(t *testing.T) {
		e := newEnv(t)
		p := e.registered(t, domain.PaymentCash)

		_, err := e.svc.VoidPurchase(t.Context(), e.actor, p.ID(), "Factura duplicada")
		require.NoError(t, err)

		events := e.audit.of("purchase.voided")
		require.Len(t, events, 1)
		require.Equal(t, p.ID(), events[0].EntityID)
		require.Equal(t, e.location, events[0].LocationID)
		require.Equal(t, map[string]string{"total": p.Totals().Total.StringFixed(2)}, events[0].Details)
	})

	t.Run("si el registro falla, la compra no se anula", func(t *testing.T) {
		e := newEnv(t)
		p := e.registered(t, domain.PaymentCash)
		e.audit.fail = errors.New("audit down")

		_, err := e.svc.VoidPurchase(t.Context(), e.actor, p.ID(), "Factura duplicada")
		require.ErrorContains(t, err, "audit down")
		stored, err := e.svc.GetPurchase(t.Context(), e.actor, p.ID())
		require.NoError(t, err)
		require.NotEqual(t, domain.PurchaseVoided, stored.Status())
	})
}
