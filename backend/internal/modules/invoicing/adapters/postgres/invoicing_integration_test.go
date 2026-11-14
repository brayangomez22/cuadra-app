//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

func TestSettingsRepository(t *testing.T) {
	f := newFixture(t)
	a, b := f.createTenant(t), f.createTenant(t)

	t.Run("un tenant sin configuración tiene la emisión apagada", func(t *testing.T) {
		s, err := f.settings.Get(t.Context(), a.id)
		require.NoError(t, err)
		require.Equal(t, domain.DefaultSettings(), s)
	})

	t.Run("guarda y reemplaza la configuración", func(t *testing.T) {
		require.NoError(t, f.settings.Save(t.Context(), a.id, domain.Settings{Enabled: true, DefaultKind: domain.KindInvoice}))
		require.NoError(t, f.settings.Save(t.Context(), a.id, domain.Settings{Enabled: true, DefaultKind: domain.KindPOS}))
		s, err := f.settings.Get(t.Context(), a.id)
		require.NoError(t, err)
		require.Equal(t, domain.Settings{Enabled: true, DefaultKind: domain.KindPOS}, s)
	})

	t.Run("aislamiento: el tenant B no ve la configuración del A", func(t *testing.T) {
		s, err := f.settings.Get(t.Context(), b.id)
		require.NoError(t, err)
		require.False(t, s.Enabled)
	})
}

func TestResolutionRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda y lista las resoluciones con su siguiente consecutivo", func(t *testing.T) {
		tn := f.createTenant(t)
		r := f.createResolution(t, tn, domain.KindPOS, "pos", 100)
		_, err := f.register(t, tn)
		require.NoError(t, err)
		rs, err := f.resolutions.List(t.Context(), tn.id)
		require.NoError(t, err)
		require.Len(t, rs, 1)
		require.Equal(t, r.Params(), rs[0].Params())
		require.Equal(t, int64(2), rs[0].Next())
	})

	t.Run("borra una resolución sin documentos y no una que ya numeró", func(t *testing.T) {
		tn := f.createTenant(t)
		used := f.createResolution(t, tn, domain.KindPOS, "POS", 100)
		unused := f.createResolution(t, tn, domain.KindInvoice, "FE", 100)
		_, err := f.register(t, tn)
		require.NoError(t, err)
		require.ErrorIs(t, f.resolutions.Delete(t.Context(), tn.id, used.ID()), domain.ErrResolutionInUse)
		require.NoError(t, f.resolutions.Delete(t.Context(), tn.id, unused.ID()))
		require.ErrorIs(t, f.resolutions.Delete(t.Context(), tn.id, unused.ID()), domain.ErrResolutionNotFound)
	})

	t.Run("dos ventas simultáneas toman consecutivos distintos sin huecos", func(t *testing.T) {
		tn := f.createTenant(t)
		f.createResolution(t, tn, domain.KindPOS, "POS", 100)
		const sales = 8
		numbers := make(chan int64, sales)
		saleIDs := make([]uuid.UUID, sales)
		for i := range saleIDs {
			saleIDs[i] = f.createSale(t, tn)
		}
		var wg sync.WaitGroup
		for _, saleID := range saleIDs {
			wg.Go(func() {
				d, err := f.registerSale(t, tn, saleID)
				if err == nil {
					numbers <- d.Number()
				}
			})
		}
		wg.Wait()
		close(numbers)
		var got []int64
		for n := range numbers {
			got = append(got, n)
		}
		slices.Sort(got)
		require.Equal(t, []int64{1, 2, 3, 4, 5, 6, 7, 8}, got)
	})

	t.Run("aislamiento: el tenant B no ve, no numera ni borra con las resoluciones del A", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		r := f.createResolution(t, a, domain.KindPOS, "POS", 100)
		rs, err := f.resolutions.List(t.Context(), b.id)
		require.NoError(t, err)
		require.Empty(t, rs)
		_, err = f.register(t, b)
		require.ErrorIs(t, err, domain.ErrNoActiveResolution)
		require.ErrorIs(t, f.resolutions.Delete(t.Context(), b.id, r.ID()), domain.ErrResolutionNotFound)
	})
}

func TestDocumentRepository(t *testing.T) {
	f := newFixture(t)

	t.Run("guarda el documento y encola su emisión", func(t *testing.T) {
		tn := f.createTenant(t)
		f.createResolution(t, tn, domain.KindPOS, "POS", 100)
		d, err := f.register(t, tn)
		require.NoError(t, err)
		require.Contains(t, f.queue.enqueued, d.ID())

		got, err := f.documents.Get(t.Context(), tn.id, d.ID())
		require.NoError(t, err)
		require.Equal(t, d.Snapshot(), got.Snapshot())
		has, err := f.documents.ExistsForSale(t.Context(), tn.id, d.SaleID())
		require.NoError(t, err)
		require.True(t, has)
	})

	t.Run("venta y trabajo de emisión se guardan juntos o ninguno", func(t *testing.T) {
		tn := f.createTenant(t)
		r := f.createResolution(t, tn, domain.KindPOS, "POS", 100)
		f.queue.err = errors.New("river down")
		defer func() { f.queue.err = nil }()
		_, err := f.register(t, tn)
		require.Error(t, err)
		page, err := f.documents.List(t.Context(), tn.id, domain.DocumentFilter{Limit: 10})
		require.NoError(t, err)
		require.Zero(t, page.Total)
		rs, err := f.resolutions.List(t.Context(), tn.id)
		require.NoError(t, err)
		require.Equal(t, r.Next(), rs[0].Next(), "el rollback devuelve el número")
	})

	t.Run("guarda los cambios de estado y lista por estado", func(t *testing.T) {
		tn := f.createTenant(t)
		f.createResolution(t, tn, domain.KindPOS, "POS", 100)
		accepted, err := f.register(t, tn)
		require.NoError(t, err)
		rejected, err := f.register(t, tn)
		require.NoError(t, err)
		require.NoError(t, accepted.BeginAttempt(now))
		require.NoError(t, accepted.Accept(domain.Acceptance{ProviderID: "p-1", Code: "cude-1", QR: "qr"}, now))
		require.NoError(t, f.documents.Save(t.Context(), accepted))
		require.NoError(t, rejected.Reject([]string{"FAD06", "FAK24"}, now))
		require.NoError(t, f.documents.Save(t.Context(), rejected))

		status := domain.StatusRejected
		page, err := f.documents.List(t.Context(), tn.id, domain.DocumentFilter{Status: &status, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total)
		require.Equal(t, rejected.Snapshot(), page.Documents[0].Snapshot())
		got, err := f.documents.Get(t.Context(), tn.id, accepted.ID())
		require.NoError(t, err)
		require.Equal(t, accepted.Snapshot(), got.Snapshot())
	})

	t.Run("reencolar guarda el reintento y vuelve a encolar", func(t *testing.T) {
		tn := f.createTenant(t)
		f.createResolution(t, tn, domain.KindPOS, "POS", 100)
		d, err := f.register(t, tn)
		require.NoError(t, err)
		require.NoError(t, d.Reject([]string{"FAD06"}, now))
		require.NoError(t, f.documents.Save(t.Context(), d))
		require.NoError(t, d.Retry(now))
		require.NoError(t, f.documents.Requeue(t.Context(), d))
		got, err := f.documents.Get(t.Context(), tn.id, d.ID())
		require.NoError(t, err)
		require.Equal(t, domain.StatusPending, got.Status())
		require.Empty(t, got.Reasons())
		require.Equal(t, 2, countOf(f.queue.enqueued, d.ID()))
	})

	t.Run("encuentra los tenants con documentos enviados y sus documentos", func(t *testing.T) {
		tn := f.createTenant(t)
		f.createResolution(t, tn, domain.KindPOS, "POS", 100)
		d, err := f.register(t, tn)
		require.NoError(t, err)
		require.NoError(t, d.MarkSent("p-1", now))
		require.NoError(t, f.documents.Save(t.Context(), d))

		tenants, err := f.documents.SentTenants(t.Context(), 1000)
		require.NoError(t, err)
		require.Contains(t, tenants, tn.id)
		sent, err := f.documents.Sent(t.Context(), tn.id, 10)
		require.NoError(t, err)
		require.Len(t, sent, 1)
		require.Equal(t, d.ID(), sent[0].ID())
	})

	t.Run("aislamiento: el tenant B no ve ni modifica los documentos del A", func(t *testing.T) {
		a, b := f.createTenant(t), f.createTenant(t)
		f.createResolution(t, a, domain.KindPOS, "POS", 100)
		d, err := f.register(t, a)
		require.NoError(t, err)

		_, err = f.documents.Get(t.Context(), b.id, d.ID())
		require.ErrorIs(t, err, domain.ErrDocumentNotFound)
		page, err := f.documents.List(t.Context(), b.id, domain.DocumentFilter{Limit: 10})
		require.NoError(t, err)
		require.Zero(t, page.Total)
		has, err := f.documents.ExistsForSale(t.Context(), b.id, d.SaleID())
		require.NoError(t, err)
		require.False(t, has)
		forged := domain.RestoreDocument(func() domain.DocumentSnapshot {
			s := d.Snapshot()
			s.TenantID = b.id
			s.Status = domain.StatusRejected
			s.Reasons = []string{"x"}
			return s
		}())
		require.ErrorIs(t, f.documents.Save(t.Context(), forged), domain.ErrDocumentNotFound)
	})

	t.Run("la app no puede borrar un documento electrónico", func(t *testing.T) {
		tn := f.createTenant(t)
		f.createResolution(t, tn, domain.KindPOS, "POS", 100)
		d, err := f.register(t, tn)
		require.NoError(t, err)
		err = f.db.WithTenantTx(t.Context(), tn.id, func(tx pgx.Tx) error {
			_, err := tx.Exec(context.Background(), "DELETE FROM electronic_documents WHERE id = $1", d.ID())
			return err
		})
		require.ErrorContains(t, err, "permission denied")
	})
}

func countOf[T comparable](s []T, v T) int {
	n := 0
	for _, x := range s {
		if x == v {
			n++
		}
	}
	return n
}
