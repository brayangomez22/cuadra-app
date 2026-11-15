package httpadapter_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

func TestGetElectronicDocumentFile(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	path := "/api/v1/invoicing/documents/" + id.String() + "/files/"

	t.Run("descarga el PDF del documento con su tipo y nombre", func(t *testing.T) {
		uc := stub{file: func(a app.Actor, got uuid.UUID, kind string) (domain.File, error) {
			require.Equal(t, owner.TenantID, a.TenantID)
			require.Equal(t, id, got)
			require.Equal(t, "pdf", kind)
			return domain.NewFile("pdf", []byte("%PDF-1.7"))
		}}

		rec := serve(t, uc, http.MethodGet, path+"pdf", "")

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
		require.Contains(t, rec.Header().Get("Content-Disposition"), id.String()+".pdf")
		require.Equal(t, "%PDF-1.7", rec.Body.String())
	})

	t.Run("descarga el XML", func(t *testing.T) {
		uc := stub{file: func(app.Actor, uuid.UUID, string) (domain.File, error) {
			return domain.NewFile("xml", []byte("<AttachedDocument/>"))
		}}
		rec := serve(t, uc, http.MethodGet, path+"xml", "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, "application/xml", rec.Header().Get("Content-Type"))
	})

	t.Run("un documento sin el archivo responde 404 document_file_not_found", func(t *testing.T) {
		uc := stub{file: func(app.Actor, uuid.UUID, string) (domain.File, error) {
			return domain.File{}, domain.ErrFileNotFound
		}}
		requireError(t, serve(t, uc, http.MethodGet, path+"pdf", ""), http.StatusNotFound, "document_file_not_found")
	})

	t.Run("un documento que no existe responde 404 document_not_found", func(t *testing.T) {
		uc := stub{file: func(app.Actor, uuid.UUID, string) (domain.File, error) {
			return domain.File{}, domain.ErrDocumentNotFound
		}}
		requireError(t, serve(t, uc, http.MethodGet, path+"xml", ""), http.StatusNotFound, "document_not_found")
	})
}
