package httpadapter

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// unauthorizedMessage answers a request that reaches the module without a
// principal, which RequireAuth should have stopped.
const unauthorizedMessage = "Tu sesión no es válida o expiró. Inicia sesión de nuevo."

// Code and message of a missing location. In a body it is a rule (422); in
// the path of GET /locations/{id}/stock it is the resource (404).
const (
	locationNotFoundCode    = "location_not_found"
	locationNotFoundMessage = "La sede no existe."
)

// errorRules maps the inventory domain errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrLocationNameTaken, Status: http.StatusConflict, Code: "location_name_taken", Message: "Ya existe una sede con ese nombre."},
	{Err: domain.ErrInsufficientStock, Status: http.StatusUnprocessableEntity, Code: "insufficient_stock", Message: "No hay stock suficiente para esta operación."},
	{Err: domain.ErrAdjustmentReasonRequired, Status: http.StatusUnprocessableEntity, Code: "adjustment_reason_required", Message: "Escribe el motivo del ajuste."},
	{Err: domain.ErrAdjustmentReasonTooLong, Status: http.StatusUnprocessableEntity, Code: "adjustment_reason_too_long", Message: "El motivo del ajuste puede tener máximo 500 caracteres."},
	{Err: domain.ErrSameLocationTransfer, Status: http.StatusUnprocessableEntity, Code: "same_location_transfer", Message: "La sede de origen y la de destino deben ser distintas."},
	{Err: domain.ErrInvalidQuantity, Status: http.StatusUnprocessableEntity, Code: "invalid_quantity", Message: "La cantidad no es válida: distinta de cero y con máximo 4 decimales."},
	{Err: domain.ErrInvalidUnitCost, Status: http.StatusUnprocessableEntity, Code: "invalid_unit_cost", Message: "El costo no puede ser negativo y tiene máximo 4 decimales."},
	{Err: domain.ErrInvalidLocationName, Status: http.StatusUnprocessableEntity, Code: "invalid_location_name", Message: "Escribe el nombre de la sede (máximo 100 caracteres)."},
	{Err: domain.ErrProductNotFound, Status: http.StatusUnprocessableEntity, Code: "product_not_found", Message: "El producto no existe."},
	{Err: domain.ErrLocationNotFound, Status: http.StatusUnprocessableEntity, Code: locationNotFoundCode, Message: locationNotFoundMessage},
	{Err: domain.ErrLocationInactive, Status: http.StatusUnprocessableEntity, Code: "location_inactive", Message: "La sede está inactiva. Actívala antes de mover mercancía en ella."},
}
