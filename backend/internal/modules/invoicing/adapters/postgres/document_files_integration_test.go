//go:build integration

package postgres_test

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

func file(t *testing.T, kind, content string) domain.File {
	t.Helper()
	f, err := domain.NewFile(kind, []byte(content))
	require.NoError(t, err)
	return f
}

func TestFileStore(t *testing.T) {
	f := newFixture(t)
	setUp := func(t *testing.T) (*tenant, *domain.Document) {
		t.Helper()
		tn := f.createTenant(t)
		f.createResolution(t, tn, domain.KindPOS, "POS", 100)
		doc, err := f.register(t, tn)
		require.NoError(t, err)
		return tn, doc
	}

	t.Run("guarda el XML y el PDF de un documento", func(t *testing.T) {
		tn, doc := setUp(t)
		require.NoError(t, f.files.Put(t.Context(), tn.id, doc.ID(), file(t, "xml", "<AttachedDocument/>")))
		require.NoError(t, f.files.Put(t.Context(), tn.id, doc.ID(), file(t, "pdf", "%PDF-1.7")))

		got, err := f.files.Get(t.Context(), tn.id, doc.ID(), domain.FilePDF)
		require.NoError(t, err)
		require.Equal(t, file(t, "pdf", "%PDF-1.7"), got)
		got, err = f.files.Get(t.Context(), tn.id, doc.ID(), domain.FileXML)
		require.NoError(t, err)
		require.Equal(t, []byte("<AttachedDocument/>"), got.Content)
	})

	t.Run("guardarlo otra vez reemplaza el archivo", func(t *testing.T) {
		tn, doc := setUp(t)
		require.NoError(t, f.files.Put(t.Context(), tn.id, doc.ID(), file(t, "pdf", "primero")))
		require.NoError(t, f.files.Put(t.Context(), tn.id, doc.ID(), file(t, "pdf", "segundo")))
		got, err := f.files.Get(t.Context(), tn.id, doc.ID(), domain.FilePDF)
		require.NoError(t, err)
		require.Equal(t, []byte("segundo"), got.Content)
	})

	t.Run("un archivo que no está es ErrFileNotFound", func(t *testing.T) {
		tn, doc := setUp(t)
		_, err := f.files.Get(t.Context(), tn.id, doc.ID(), domain.FilePDF)
		require.ErrorIs(t, err, domain.ErrFileNotFound)
	})

	t.Run("la aplicación no puede borrar los archivos", func(t *testing.T) {
		tn, doc := setUp(t)
		require.NoError(t, f.files.Put(t.Context(), tn.id, doc.ID(), file(t, "pdf", "%PDF-1.7")))
		err := f.db.WithTenantTx(t.Context(), tn.id, func(tx pgx.Tx) error {
			_, err := tx.Exec(t.Context(), "DELETE FROM electronic_document_files WHERE document_id = $1", doc.ID())
			return err
		})
		require.ErrorContains(t, err, "permission denied")
	})

	t.Run("aislamiento: el tenant B no ve ni guarda archivos de los documentos del tenant A", func(t *testing.T) {
		a, doc := setUp(t)
		b := f.createTenant(t)
		require.NoError(t, f.files.Put(t.Context(), a.id, doc.ID(), file(t, "pdf", "%PDF-1.7")))

		_, err := f.files.Get(t.Context(), b.id, doc.ID(), domain.FilePDF)
		require.ErrorIs(t, err, domain.ErrFileNotFound)
		require.Error(t, f.files.Put(t.Context(), b.id, doc.ID(), file(t, "pdf", "ajeno")))
	})
}
