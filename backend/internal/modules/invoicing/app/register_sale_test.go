package app_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

func TestRegisterSale(t *testing.T) {
	t.Run("con la emisión desactivada la venta no crea documento", func(t *testing.T) {
		e := newEnv(t)
		e.resolution(t, domain.KindPOS, "POS", 1, 100)
		saleID := e.sale(nil)
		require.NoError(t, e.svc.RegisterSale(t.Context(), e.actor, app.SaleDocumentRequest{SaleID: saleID}))
		has, err := e.svc.HasDocument(t.Context(), e.actor, saleID)
		require.NoError(t, err)
		require.False(t, has)
		require.Empty(t, e.store.enqueued)
	})

	t.Run("numera la venta con la resolución vigente y encola su emisión sin llamar al proveedor", func(t *testing.T) {
		e := newEnv(t)
		e.enable(t, domain.KindPOS)
		e.resolution(t, domain.KindPOS, "POS", 41, 100)
		first, second := e.register(t, e.sale(nil)), e.register(t, e.sale(nil))

		require.Equal(t, domain.StatusPending, first.Status())
		require.Equal(t, domain.KindPOS, first.Kind())
		require.Equal(t, "POS", first.Prefix())
		require.Equal(t, []int64{41, 42}, []int64{first.Number(), second.Number()})
		require.Equal(t, e.now, first.IssuedAt())
		require.Equal(t, e.actor.UserID, first.CreatedBy())
		require.Equal(t, []any{first.ID(), second.ID()}, []any{e.store.enqueued[0], e.store.enqueued[1]})
		require.Empty(t, e.invoicer.issued, "la venta se confirma aunque el proveedor esté caído")
		require.Equal(t, int64(2), e.counter(t, "cuadra.invoicing.documents", "document.status", "pending"))
		require.Equal(t, second.ID().String(), e.span(t, "invoicing.RegisterSale")["document.id"].AsString())
	})

	t.Run("venta a crédito toma el número de la resolución de facturas", func(t *testing.T) {
		e := newEnv(t)
		e.enable(t, domain.KindPOS)
		e.resolution(t, domain.KindPOS, "POS", 1, 100)
		e.resolution(t, domain.KindInvoice, "FE", 500, 600)
		customer := newID()
		saleID := e.sale(&customer)
		err := e.svc.RegisterSale(t.Context(), e.actor, app.SaleDocumentRequest{SaleID: saleID, Credit: true, HasCustomer: true})
		require.NoError(t, err)
		d := e.documentOf(t, saleID)
		require.Equal(t, domain.KindInvoice, d.Kind())
		require.Equal(t, int64(500), d.Number())
	})

	t.Run("sin resolución vigente la venta se rechaza y no guarda nada", func(t *testing.T) {
		e := newEnv(t)
		e.enable(t, domain.KindPOS)
		e.resolution(t, domain.KindInvoice, "FE", 1, 100)
		err := e.svc.RegisterSale(t.Context(), e.actor, app.SaleDocumentRequest{SaleID: e.sale(nil)})
		require.ErrorIs(t, err, domain.ErrNoActiveResolution)
		require.Empty(t, e.store.documents)
		require.Empty(t, e.store.enqueued)
	})

	t.Run("factura electrónica pedida sin cliente se rechaza", func(t *testing.T) {
		e := newEnv(t)
		e.enable(t, domain.KindPOS)
		e.resolution(t, domain.KindInvoice, "FE", 1, 100)
		err := e.svc.RegisterSale(t.Context(), e.actor, app.SaleDocumentRequest{SaleID: e.sale(nil), Kind: "invoice"})
		require.ErrorIs(t, err, domain.ErrInvoiceRequiresCustomer)
	})

	t.Run("avisa una sola vez cuando la numeración cruza el umbral de fin de rango", func(t *testing.T) {
		e := newEnv(t)
		e.enable(t, domain.KindPOS)
		e.resolution(t, domain.KindPOS, "POS", 1, 20) // el aviso llega con 2 consecutivos libres
		for range 17 {
			e.register(t, e.sale(nil))
		}
		require.Zero(t, e.counter(t, "cuadra.invoicing.numbering_near_end", "", ""))
		e.register(t, e.sale(nil))
		e.register(t, e.sale(nil))
		require.Equal(t, int64(1), e.counter(t, "cuadra.invoicing.numbering_near_end", "document.kind", "pos"))
	})

	t.Run("la resolución vencida no numera aunque le queden consecutivos", func(t *testing.T) {
		e := newEnv(t)
		e.enable(t, domain.KindPOS)
		e.resolution(t, domain.KindPOS, "POS", 1, 100)
		e.now = time.Date(2027, time.January, 2, 15, 0, 0, 0, time.UTC)
		err := e.svc.RegisterSale(t.Context(), e.actor, app.SaleDocumentRequest{SaleID: e.sale(nil)})
		require.ErrorIs(t, err, domain.ErrNoActiveResolution)
	})
}
