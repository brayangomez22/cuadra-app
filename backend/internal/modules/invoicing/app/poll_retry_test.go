package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// sent leaves a document sent: the DIAN was not available.
func (e *env) sent(t *testing.T) *domain.Document {
	t.Helper()
	d := e.pending(t)
	e.invoicer.result = app.Result{Status: domain.StatusSent, ProviderID: "prov-1"}
	require.NoError(t, e.issue(t, d))
	return d
}

func (e *env) ref(d *domain.Document) app.DocumentRef {
	return app.DocumentRef{TenantID: d.TenantID(), Kind: d.Kind(), Prefix: d.Prefix(), Number: d.Number()}
}

func TestPollSent(t *testing.T) {
	t.Run("contingencia: el documento enviado se acepta en la consulta periódica", func(t *testing.T) {
		e := newEnv(t)
		d := e.sent(t)
		e.invoicer.received[e.ref(d)] = app.Result{Status: domain.StatusAccepted, ProviderID: "prov-1", Code: "cude-7", QR: "qr"}

		n, err := e.svc.PollSent(t.Context())
		require.NoError(t, err)
		require.Equal(t, 1, n)
		got := e.reload(t, d.ID())
		require.Equal(t, domain.StatusAccepted, got.Status())
		require.Equal(t, "cude-7", got.Code())
		require.Len(t, e.invoicer.issued, 1, "la consulta no reenvía")
	})

	t.Run("deja enviado el documento que la DIAN todavía no responde", func(t *testing.T) {
		e := newEnv(t)
		d := e.sent(t)
		n, err := e.svc.PollSent(t.Context())
		require.NoError(t, err)
		require.Zero(t, n)
		require.Equal(t, domain.StatusSent, e.reload(t, d.ID()).Status())
	})

	t.Run("un rechazo en la consulta guarda sus motivos", func(t *testing.T) {
		e := newEnv(t)
		d := e.sent(t)
		e.invoicer.received[e.ref(d)] = app.Result{Status: domain.StatusRejected, Reasons: []string{"FAJ43"}}
		_, err := e.svc.PollSent(t.Context())
		require.NoError(t, err)
		require.Equal(t, []string{"FAJ43"}, e.reload(t, d.ID()).Reasons())
	})

	t.Run("con el proveedor caído sigue con los demás y devuelve el error", func(t *testing.T) {
		e := newEnv(t)
		e.sent(t)
		e.invoicer.lookupErr = &app.ProviderError{Reason: app.ReasonTimeout, Err: errTimeout}
		_, err := e.svc.PollSent(t.Context())
		require.Error(t, err)
	})
}

func TestRetryDocument(t *testing.T) {
	t.Run("reintento manual de un rechazado lo vuelve a encolar con el mismo número", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		e.invoicer.result = app.Result{Status: domain.StatusRejected, Reasons: []string{"FAK24"}}
		require.NoError(t, e.issue(t, d))

		got, err := e.svc.RetryDocument(t.Context(), e.actor, d.ID())
		require.NoError(t, err)
		require.Equal(t, domain.StatusPending, got.Status())
		require.Empty(t, got.Reasons())
		require.Equal(t, d.Number(), got.Number())
		require.Equal(t, []any{d.ID(), d.ID()}, []any{e.store.enqueued[0], e.store.enqueued[1]})
		require.Equal(t, int64(1), e.counter(t, "cuadra.invoicing.manual_retries", "document.status", "rejected"))

		e.invoicer.result = app.Result{Status: domain.StatusAccepted, Code: "cude-2"}
		require.NoError(t, e.issue(t, got))
		require.Equal(t, domain.StatusAccepted, e.reload(t, d.ID()).Status())
	})

	t.Run("no reintenta un documento aceptado", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		require.NoError(t, e.issue(t, d))
		_, err := e.svc.RetryDocument(t.Context(), e.actor, d.ID())
		require.ErrorIs(t, err, domain.ErrDocumentNotRetryable)
	})

	t.Run("aislamiento: no reintenta un documento de otro tenant", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		other := app.Actor{TenantID: newID(), UserID: newID()}
		_, err := e.svc.RetryDocument(t.Context(), other, d.ID())
		require.ErrorIs(t, err, domain.ErrDocumentNotFound)
	})
}
