package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

var errTimeout = errors.New("timeout")

func TestResolutions(t *testing.T) {
	t.Run("crea y lista las resoluciones con su aviso de fin de rango", func(t *testing.T) {
		e := newEnv(t)
		e.resolution(t, domain.KindPOS, "pos", 1, 1000)
		e.resolution(t, domain.KindInvoice, "FE", 1, 10)
		e.enable(t, domain.KindInvoice)
		for range 9 {
			e.register(t, e.sale(nil))
		}

		views, err := e.svc.ListResolutions(t.Context(), e.actor)
		require.NoError(t, err)
		require.Len(t, views, 2)
		require.Equal(t, "FE", views[0].Resolution.Params().Prefix)
		require.True(t, views[0].NearEnd, "le queda un consecutivo de diez")
		require.Equal(t, "POS", views[1].Resolution.Params().Prefix)
		require.False(t, views[1].NearEnd)
		require.NotEmpty(t, e.span(t, "invoicing.ListResolutions")["tenant.id"].AsString())
	})

	t.Run("rechaza una resolución que se cruza con otra del mismo prefijo", func(t *testing.T) {
		e := newEnv(t)
		e.resolution(t, domain.KindPOS, "POS", 1, 1000)
		_, err := e.svc.CreateResolution(t.Context(), e.actor, domain.ResolutionParams{
			Kind: domain.KindPOS, Prefix: "pos", Number: "18764000009999", From: 900, To: 2000,
			ValidFrom: domain.NewDate(2027, time.January, 1), ValidTo: domain.NewDate(2027, time.December, 31),
		})
		require.ErrorIs(t, err, domain.ErrResolutionOverlaps)
	})

	t.Run("borra una resolución sin documentos y no una que ya numeró", func(t *testing.T) {
		e := newEnv(t)
		unused := e.resolution(t, domain.KindInvoice, "FE", 1, 100)
		e.pending(t)
		views, err := e.svc.ListResolutions(t.Context(), e.actor)
		require.NoError(t, err)
		var used *domain.Resolution
		for _, v := range views {
			if v.Resolution.Used() {
				used = v.Resolution
			}
		}
		require.NotNil(t, used)

		require.NoError(t, e.svc.DeleteResolution(t.Context(), e.actor, unused))
		require.ErrorIs(t, e.svc.DeleteResolution(t.Context(), e.actor, used.ID()), domain.ErrResolutionInUse)
	})

	t.Run("aislamiento: otro tenant no ve ni borra las resoluciones", func(t *testing.T) {
		e := newEnv(t)
		id := e.resolution(t, domain.KindPOS, "POS", 1, 100)
		other := app.Actor{TenantID: newID(), UserID: newID()}
		views, err := e.svc.ListResolutions(t.Context(), other)
		require.NoError(t, err)
		require.Empty(t, views)
		require.ErrorIs(t, e.svc.DeleteResolution(t.Context(), other, id), domain.ErrResolutionNotFound)
	})
}

func TestSettings(t *testing.T) {
	t.Run("un tenant nuevo tiene la emisión apagada", func(t *testing.T) {
		e := newEnv(t)
		s, err := e.svc.GetSettings(t.Context(), e.actor)
		require.NoError(t, err)
		require.Equal(t, domain.DefaultSettings(), s)
	})

	t.Run("activa la emisión con factura por defecto", func(t *testing.T) {
		e := newEnv(t)
		s, err := e.svc.UpdateSettings(t.Context(), e.actor, true, "invoice")
		require.NoError(t, err)
		require.Equal(t, domain.Settings{Enabled: true, DefaultKind: domain.KindInvoice}, s)
		got, err := e.svc.GetSettings(t.Context(), e.actor)
		require.NoError(t, err)
		require.Equal(t, s, got)
	})

	t.Run("rechaza un tipo por defecto desconocido", func(t *testing.T) {
		e := newEnv(t)
		_, err := e.svc.UpdateSettings(t.Context(), e.actor, true, "remision")
		require.ErrorIs(t, err, domain.ErrInvalidKind)
	})
}

func TestDocuments(t *testing.T) {
	t.Run("filtra los documentos por estado", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		e.register(t, e.sale(nil))
		require.NoError(t, e.issue(t, d))
		accepted := domain.StatusAccepted
		page, err := e.svc.ListDocuments(t.Context(), e.actor, domain.DocumentFilter{Status: &accepted, Limit: 10})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total)
		require.Equal(t, d.ID(), page.Documents[0].ID())
	})

	t.Run("aislamiento: otro tenant no ve el documento", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		_, err := e.svc.GetDocument(t.Context(), app.Actor{TenantID: newID(), UserID: newID()}, d.ID())
		require.ErrorIs(t, err, domain.ErrDocumentNotFound)
	})
}
