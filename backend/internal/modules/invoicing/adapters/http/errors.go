package httpadapter

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// errorRules maps the invoicing domain errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrResolutionNotFound, Status: http.StatusNotFound, Code: "resolution_not_found", Message: "La resolución de numeración no existe."},
	{Err: domain.ErrDocumentNotFound, Status: http.StatusNotFound, Code: "document_not_found", Message: "El documento electrónico no existe."},
	{Err: domain.ErrResolutionOverlaps, Status: http.StatusConflict, Code: "resolution_overlaps", Message: "El rango se cruza con el de otra resolución del mismo tipo y prefijo."},
	{Err: domain.ErrResolutionInUse, Status: http.StatusConflict, Code: "resolution_in_use", Message: "La resolución ya numeró documentos y no se puede borrar."},
	{Err: domain.ErrDocumentNotRetryable, Status: http.StatusConflict, Code: "document_not_retryable", Message: "Solo se reenvía un documento pendiente o rechazado."},
	{Err: domain.ErrInvalidKind, Status: http.StatusUnprocessableEntity, Code: "invalid_document_kind", Message: "El tipo de documento debe ser documento POS o factura electrónica."},
	{Err: domain.ErrInvalidPrefix, Status: http.StatusUnprocessableEntity, Code: "invalid_prefix", Message: "El prefijo tiene hasta 4 letras o dígitos."},
	{Err: domain.ErrInvalidResolutionNumber, Status: http.StatusUnprocessableEntity, Code: "invalid_resolution_number", Message: "El número de la resolución lleva solo dígitos."},
	{Err: domain.ErrInvalidTechnicalKey, Status: http.StatusUnprocessableEntity, Code: "invalid_technical_key", Message: "La clave técnica tiene hasta 100 caracteres."},
	{Err: domain.ErrInvalidRange, Status: http.StatusUnprocessableEntity, Code: "invalid_range", Message: "El rango empieza en 1 o más y termina en un número igual o mayor."},
	{Err: domain.ErrInvalidValidity, Status: http.StatusUnprocessableEntity, Code: "invalid_validity", Message: "La vigencia termina el mismo día que empieza o después."},
}
