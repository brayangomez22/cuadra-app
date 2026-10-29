package app_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

func TestReceiveCash(t *testing.T) {
	t.Run("el efectivo de un abono entra como ingreso a la caja abierta del usuario", func(t *testing.T) {
		e := newEnv(t)
		session := e.openCash(t, newID(), "50000")
		id, err := e.svc.ReceiveCash(t.Context(), e.actor, dec("30000"), "Abono de cartera")
		require.NoError(t, err)
		require.Equal(t, session.ID(), id)
		require.Len(t, e.store.movements, 1)
		m := e.store.movements[0]
		require.Equal(t, domain.CashIn, m.Kind())
		requireDec(t, "30000", m.Amount())
		require.Equal(t, "Abono de cartera", m.Reason())
		_, attrs := e.span(t, "sales.ReceiveCash")
		require.Equal(t, session.ID().String(), attrs["cash_session.id"].AsString())
	})

	t.Run("sin caja abierta no registra nada si el tenant permite trabajar sin caja", func(t *testing.T) {
		e := newEnv(t)
		id, err := e.svc.ReceiveCash(t.Context(), e.actor, dec("30000"), "Abono de cartera")
		require.NoError(t, err)
		require.Equal(t, uuid.Nil, id)
		require.Empty(t, e.store.movements)
	})

	t.Run("sin caja abierta rechaza el efectivo si el tenant exige caja", func(t *testing.T) {
		e := newEnv(t)
		e.requireCashSession()
		_, err := e.svc.ReceiveCash(t.Context(), e.actor, dec("30000"), "Abono de cartera")
		require.ErrorIs(t, err, domain.ErrCashSessionRequired)
	})
}
