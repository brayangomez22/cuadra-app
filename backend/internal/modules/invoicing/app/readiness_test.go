package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

func TestReadiness(t *testing.T) {
	t.Run("sin la emisión activa no está lista", func(t *testing.T) {
		e := newEnv(t)
		e.resolution(t, domain.KindPOS, "POS", 1, 100)

		r, err := e.svc.Readiness(t.Context(), e.actor)
		require.NoError(t, err)
		require.Equal(t, app.Readiness{}, r)
		require.NotEmpty(t, e.span(t, "invoicing.Readiness")["tenant.id"].AsString())
	})
	t.Run("está lista con la emisión activa y una resolución vigente del tipo por defecto", func(t *testing.T) {
		e := newEnv(t)
		e.resolution(t, domain.KindInvoice, "FE", 1, 100)
		e.enable(t, domain.KindInvoice)

		r, err := e.svc.Readiness(t.Context(), e.actor)
		require.NoError(t, err)
		require.Equal(t, app.Readiness{Enabled: true, Ready: true}, r)
	})
	t.Run("no está lista con una resolución de otro tipo", func(t *testing.T) {
		e := newEnv(t)
		e.resolution(t, domain.KindInvoice, "FE", 1, 100)
		e.enable(t, domain.KindPOS)

		r, err := e.svc.Readiness(t.Context(), e.actor)
		require.NoError(t, err)
		require.Equal(t, app.Readiness{Enabled: true}, r)
	})
	t.Run("no está lista con la resolución vencida", func(t *testing.T) {
		e := newEnv(t)
		e.resolution(t, domain.KindPOS, "POS", 1, 100)
		e.enable(t, domain.KindPOS)
		e.now = time.Date(2027, time.January, 2, 15, 0, 0, 0, time.UTC)

		r, err := e.svc.Readiness(t.Context(), e.actor)
		require.NoError(t, err)
		require.False(t, r.Ready)
	})
	t.Run("no está lista con la resolución agotada", func(t *testing.T) {
		e := newEnv(t)
		e.resolution(t, domain.KindPOS, "POS", 1, 2)
		e.enable(t, domain.KindPOS)
		e.register(t, e.sale(nil))
		e.register(t, e.sale(nil))

		r, err := e.svc.Readiness(t.Context(), e.actor)
		require.NoError(t, err)
		require.False(t, r.Ready)
	})
}
