package httpadapter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/adapters/http/importsapi"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
)

// errUnsupportedFormat reports a file that is neither .xlsx nor .csv.
var errUnsupportedFormat = errors.New("imports: unsupported file format")

// UploadImport reads the multipart form and validates the file.
func (h *Handler) UploadImport(ctx context.Context, req importsapi.UploadImportRequestObject) (importsapi.UploadImportResponseObject, error) {
	in, err := readUpload(req.Body)
	if err != nil {
		return nil, err
	}
	preview, err := h.uc.UploadImport(ctx, actor(ctx), in)
	if err != nil {
		return nil, err
	}
	return importsapi.UploadImport201JSONResponse(previewBody(preview)), nil
}

// readUpload reads the form's parts; the contract already checked that the
// file and the location are there and the location is a UUID.
func readUpload(form *multipart.Reader) (app.UploadInput, error) {
	in := app.UploadInput{Policy: domain.UpdateExisting}
	for {
		part, err := form.NextPart()
		if errors.Is(err, io.EOF) {
			return in, nil
		}
		if err != nil {
			return app.UploadInput{}, fmt.Errorf("%w: %w", errInvalidForm, err)
		}
		err = readPart(part, &in)
		_ = part.Close()
		if err != nil {
			return app.UploadInput{}, err
		}
	}
}

// errInvalidForm reports a multipart body that cannot be read.
var errInvalidForm = errors.New("imports: invalid upload form")

func readPart(part *multipart.Part, in *app.UploadInput) error {
	// One byte over the limit tells a file that is too large.
	data, err := io.ReadAll(io.LimitReader(part, domain.MaxFileBytes+1))
	if err != nil {
		return fmt.Errorf("%w: %w", errInvalidForm, err)
	}
	switch part.FormName() {
	case "file":
		in.FileName, in.Data = part.FileName(), data
		in.Format, err = formatOf(part.FileName())
		return err
	case "location_id":
		in.LocationID, err = uuid.Parse(strings.TrimSpace(string(data)))
		if err != nil {
			return fmt.Errorf("%w: %w", errInvalidForm, err)
		}
	case "on_existing_sku":
		in.Policy = domain.ExistingPolicy(strings.TrimSpace(string(data)))
	}
	return nil
}

// formatOf tells the file's format by its extension.
func formatOf(name string) (domain.Format, error) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".xlsx":
		return domain.FormatXLSX, nil
	case ".csv":
		return domain.FormatCSV, nil
	}
	return "", errUnsupportedFormat
}
