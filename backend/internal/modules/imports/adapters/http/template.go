package httpadapter

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/http/importsapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// contentTypes are the media types of the template's formats.
var contentTypes = map[domain.Format]string{
	domain.FormatXLSX: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	domain.FormatCSV:  "text/csv; charset=utf-8",
}

// GetImportTemplate returns the tenant's template as a download.
func (h *Handler) GetImportTemplate(ctx context.Context, req importsapi.GetImportTemplateRequestObject) (importsapi.GetImportTemplateResponseObject, error) {
	format := domain.FormatXLSX
	if req.Params.Format != nil {
		format = domain.Format(*req.Params.Format)
	}
	data, err := h.uc.ImportTemplate(ctx, actor(ctx), format)
	if err != nil {
		return nil, err
	}
	return templateDownload{format: format, data: data}, nil
}

// templateDownload writes the template with a file name, so the browser
// saves it instead of showing it.
type templateDownload struct {
	format domain.Format
	data   []byte
}

func (d templateDownload) VisitGetImportTemplateResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", contentTypes[d.format])
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="plantilla-productos.%s"`, d.format))
	w.Header().Set("Content-Length", strconv.Itoa(len(d.data)))
	w.WriteHeader(http.StatusOK)
	_, err := bytes.NewReader(d.data).WriteTo(w)
	return err
}
