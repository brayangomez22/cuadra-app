package app_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// files are the XML and the PDF the provider returns with an acceptance.
func files(t *testing.T) []domain.File {
	t.Helper()
	xml, err := domain.NewFile("xml", []byte("<AttachedDocument/>"))
	require.NoError(t, err)
	pdf, err := domain.NewFile("pdf", []byte("%PDF-1.7 tirilla"))
	require.NoError(t, err)
	return []domain.File{xml, pdf}
}

func TestIssueDocumentProvider(t *testing.T) {
	t.Run("un documento del tenant demo no llega al proveedor", func(t *testing.T) {
		e := newEnv(t)
		e.issuers.demo(e.actor.TenantID)
		d := e.pending(t)
		require.NoError(t, e.issue(t, d))

		require.Empty(t, e.invoicer.issued)
		require.Len(t, e.demoInvoicer.issued, 1)
		require.Equal(t, domain.StatusAccepted, e.reload(t, d.ID()).Status())
	})

	t.Run("el demo tampoco consulta al proveedor al reintentar ni al revisar los enviados", func(t *testing.T) {
		e := newEnv(t)
		e.issuers.demo(e.actor.TenantID)
		e.demoInvoicer.result = app.Result{Status: domain.StatusSent, ProviderID: "sim-1"}
		d := e.pending(t)
		require.NoError(t, e.issue(t, d))
		_, err := e.svc.PollSent(t.Context())
		require.NoError(t, err)

		require.Zero(t, e.invoicer.lookups)
		require.Equal(t, 1, e.demoInvoicer.lookups)
	})

	t.Run("un tenant real va al proveedor configurado", func(t *testing.T) {
		e := newEnv(t)
		require.NoError(t, e.issue(t, e.pending(t)))
		require.Len(t, e.invoicer.issued, 1)
		require.Empty(t, e.demoInvoicer.issued)
	})

	t.Run("el documento lleva al emisor con su municipio DANE", func(t *testing.T) {
		cases := []struct {
			name   string
			issuer domain.Issuer
		}{
			{"ferretería de Medellín", domain.Issuer{NIT: "900123456-1", Name: "Ferretería El Tornillo", Address: "Cra 50 # 40-20", City: "Medellín", MunicipalityCode: "05001"}},
			{"almacén de ropa de Bogotá", domain.Issuer{NIT: "800987654-3", Name: "Moda Andina", Address: "Cl 72 # 10-15", City: "Bogotá", MunicipalityCode: "11001"}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				e := newEnv(t)
				e.issuers.issuers[e.actor.TenantID] = tc.issuer
				require.NoError(t, e.issue(t, e.pending(t)))
				require.Equal(t, tc.issuer, e.invoicer.issued[0].Issuer)
			})
		}
	})

	t.Run("el XML y el PDF de un documento aceptado quedan en el almacenamiento", func(t *testing.T) {
		e := newEnv(t)
		e.invoicer.result.Files = files(t)
		d := e.pending(t)
		require.NoError(t, e.issue(t, d))

		pdf, err := e.svc.DocumentFile(t.Context(), e.actor, d.ID(), "pdf")
		require.NoError(t, err)
		require.Equal(t, []byte("%PDF-1.7 tirilla"), pdf.Content)
		xml, err := e.svc.DocumentFile(t.Context(), e.actor, d.ID(), "xml")
		require.NoError(t, err)
		require.Equal(t, "application/xml", xml.Kind.ContentType())
		require.Equal(t, int64(1), e.counter(t, "cuadra.invoicing.files", "file.kind", "pdf"))
		require.Equal(t, d.ID().String(), e.span(t, "invoicing.GetDocumentFile")["document.id"].AsString())
	})

	t.Run("si no se pueden guardar los archivos el documento sigue pendiente para reintento", func(t *testing.T) {
		e := newEnv(t)
		e.invoicer.result.Files = files(t)
		e.store.filesDown = true
		d := e.pending(t)
		require.Error(t, e.issue(t, d))
		require.Equal(t, domain.StatusPending, e.reload(t, d.ID()).Status())
	})

	t.Run("un documento sin archivos responde que no los tiene", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		require.NoError(t, e.issue(t, d))
		_, err := e.svc.DocumentFile(t.Context(), e.actor, d.ID(), "pdf")
		require.ErrorIs(t, err, domain.ErrFileNotFound)
	})

	t.Run("aislamiento: otro tenant no ve los archivos", func(t *testing.T) {
		e := newEnv(t)
		e.invoicer.result.Files = files(t)
		d := e.pending(t)
		require.NoError(t, e.issue(t, d))
		other := app.Actor{TenantID: newID(), UserID: newID()}
		_, err := e.svc.DocumentFile(t.Context(), other, d.ID(), "pdf")
		require.ErrorIs(t, err, domain.ErrDocumentNotFound)
	})

	t.Run("rechaza un tipo de archivo desconocido", func(t *testing.T) {
		e := newEnv(t)
		d := e.pending(t)
		_, err := e.svc.DocumentFile(t.Context(), e.actor, d.ID(), "zip")
		require.ErrorIs(t, err, domain.ErrInvalidFileKind)
	})
}
