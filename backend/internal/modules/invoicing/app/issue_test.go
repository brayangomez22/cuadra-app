package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// pending registers a POS document of a sale without customer.
func (e *env) pending(t *testing.T) *domain.Document {
	t.Helper()
	e.enable(t, domain.KindPOS)
	e.resolution(t, domain.KindPOS, "POS", 1, 1000)
	return e.register(t, e.sale(nil))
}

func (e *env) issue(t *testing.T, d *domain.Document) error {
	t.Helper()
	return e.svc.IssueDocument(t.Context(), e.actor.TenantID, d.ID())
}

func TestIssueDocument(t *testing.T) {
	t.Run("aceptado guarda el CUDE y el QR y mide la latencia", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		e.now = e.now.Add(4 * time.Second)
		require.NoError(t, e.issue(t, d))

		got := e.reload(t, d.ID())
		require.Equal(t, domain.StatusAccepted, got.Status())
		require.Equal(t, "cude-1", got.Code())
		require.Equal(t, "https://qr", got.QR())
		require.Equal(t, 1, got.Attempts())
		require.Equal(t, int64(1), e.counter(t, "cuadra.invoicing.documents", "document.status", "accepted"))
		require.Equal(t, uint64(1), e.histogramCount(t, "cuadra.invoicing.acceptance_latency"))
		require.Equal(t, d.ID().String(), e.span(t, "invoicing.IssueDocument")["document.id"].AsString())
	})

	t.Run("venta sin cliente se envía a consumidor final 222222222222", func(t *testing.T) {
		e := newEnv(t)
		require.NoError(t, e.issue(t, e.pending(t)))
		req := e.invoicer.issued[0]
		require.True(t, req.Buyer.IsFinalConsumer())
		require.Equal(t, "222222222222", req.Buyer.DocumentNumber)
		require.Equal(t, "POS", req.Prefix)
		require.Equal(t, int64(1), req.Number)
		require.Len(t, req.Sale.Lines, 2)
	})

	t.Run("venta con cliente se envía con sus datos", func(t *testing.T) {
		e := newEnv(t)
		e.pending(t)
		customer := newID()
		e.buyers.buyers[customer] = domain.Buyer{CustomerID: &customer, DocumentType: "nit", DocumentNumber: "900123456-7", Name: "Comercial Andina SAS"}
		saleID := e.sale(&customer)
		require.NoError(t, e.svc.RegisterSale(t.Context(), e.actor, app.SaleDocumentRequest{SaleID: saleID, HasCustomer: true}))
		require.NoError(t, e.issue(t, e.documentOf(t, saleID)))
		require.Equal(t, "Comercial Andina SAS", e.invoicer.issued[0].Buyer.Name)
	})

	t.Run("con el proveedor caído el documento queda pendiente para reintento", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		e.invoicer.down()
		err := e.issue(t, d)
		var perr *app.ProviderError
		require.ErrorAs(t, err, &perr, "el trabajo devuelve el error para que River reintente")
		got := e.reload(t, d.ID())
		require.Equal(t, domain.StatusPending, got.Status())
		require.Equal(t, 1, got.Attempts())
		require.Empty(t, got.LastError())
		require.Equal(t, int64(1), e.counter(t, "cuadra.invoicing.provider_errors", "provider.reason", "server"))
	})

	t.Run("antes de reenviar consulta por prefijo y consecutivo y no emite dos veces", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		// El proveedor recibió el documento, pero la respuesta se perdió.
		e.invoicer.received[app.DocumentRef{TenantID: d.TenantID(), Kind: d.Kind(), Prefix: d.Prefix(), Number: d.Number()}] =
			app.Result{Status: domain.StatusAccepted, ProviderID: "prov-9", Code: "cude-9"}
		e.invoicer.err = context.DeadlineExceeded
		require.Error(t, e.issue(t, d))
		e.invoicer.err = nil

		require.NoError(t, e.issue(t, d))
		require.Empty(t, e.invoicer.issued, "no lo vuelve a enviar")
		require.Equal(t, 1, e.invoicer.lookups)
		got := e.reload(t, d.ID())
		require.Equal(t, domain.StatusAccepted, got.Status())
		require.Equal(t, "cude-9", got.Code())
		require.Equal(t, int64(1), got.Number(), "un reintento nunca toma un consecutivo nuevo")
	})

	t.Run("si la consulta no lo encuentra lo envía con el mismo número", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		e.invoicer.down()
		require.Error(t, e.issue(t, d))
		e.invoicer.err = nil
		require.NoError(t, e.issue(t, d))
		require.Len(t, e.invoicer.issued, 1)
		require.Equal(t, int64(1), e.invoicer.issued[0].Number)
		require.Equal(t, 2, e.reload(t, d.ID()).Attempts())
	})

	t.Run("un rechazo guarda el motivo y no se reintenta automáticamente", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		e.invoicer.result = app.Result{Status: domain.StatusRejected, ProviderID: "prov-1", Reasons: []string{"FAK24: falta el correo del adquiriente"}}
		require.NoError(t, e.issue(t, d), "un rechazo no es un error del trabajo")
		require.NoError(t, e.issue(t, d))

		got := e.reload(t, d.ID())
		require.Equal(t, domain.StatusRejected, got.Status())
		require.Equal(t, []string{"FAK24: falta el correo del adquiriente"}, got.Reasons())
		require.Len(t, e.invoicer.issued, 1, "el segundo trabajo no lo reenvía")
		require.Equal(t, int64(1), e.counter(t, "cuadra.invoicing.documents", "document.status", "rejected"))
	})

	t.Run("contingencia de la DIAN deja el documento enviado", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		e.invoicer.result = app.Result{Status: domain.StatusSent, ProviderID: "prov-1"}
		require.NoError(t, e.issue(t, d))
		require.Equal(t, domain.StatusSent, e.reload(t, d.ID()).Status())
	})

	t.Run("una petición que el proveedor rechaza queda pendiente con su error y no se reintenta sola", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		e.invoicer.err = &app.ProviderError{Reason: app.ReasonBadRequest, Err: errors.New("422: invalid identification")}
		require.NoError(t, e.issue(t, d))
		got := e.reload(t, d.ID())
		require.Equal(t, domain.StatusPending, got.Status())
		require.Contains(t, got.LastError(), "invalid identification")

		e.invoicer.err = nil
		require.NoError(t, e.issue(t, d))
		require.Empty(t, e.invoicer.issued)
		require.Equal(t, int64(1), e.counter(t, "cuadra.invoicing.provider_errors", "provider.reason", "bad_request"))
	})

	t.Run("un documento aceptado no se vuelve a enviar", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		require.NoError(t, e.issue(t, d))
		require.NoError(t, e.issue(t, d))
		require.Len(t, e.invoicer.issued, 1)
	})
}
