package httpadapter

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/adapters/http/invoicingapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// FileUseCases are the use cases of the documents' files.
type FileUseCases interface {
	DocumentFile(ctx context.Context, a app.Actor, documentID uuid.UUID, kind string) (domain.File, error)
}

// fileHandlers are the downloads of the documents' files.
type fileHandlers struct{ uc FileUseCases }

// GetElectronicDocumentFile returns the XML or the PDF of a document.
func (h *fileHandlers) GetElectronicDocumentFile(ctx context.Context, req invoicingapi.GetElectronicDocumentFileRequestObject) (invoicingapi.GetElectronicDocumentFileResponseObject, error) {
	f, err := h.uc.DocumentFile(ctx, actor(ctx), req.DocumentId, req.FileKind)
	if err != nil {
		return nil, err
	}
	return fileDownload{documentID: req.DocumentId, file: f}, nil
}

// fileDownload writes the file with a name, so the browser saves it.
type fileDownload struct {
	documentID uuid.UUID
	file       domain.File
}

func (d fileDownload) VisitGetElectronicDocumentFileResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", d.file.Kind.ContentType())
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, d.documentID, d.file.Kind))
	w.Header().Set("Content-Length", strconv.Itoa(len(d.file.Content)))
	w.WriteHeader(http.StatusOK)
	_, err := bytes.NewReader(d.file.Content).WriteTo(w)
	return err
}
