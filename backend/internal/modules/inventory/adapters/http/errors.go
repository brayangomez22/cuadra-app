package httpadapter

import (
	"errors"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// Code and message of a missing location. In a body it is a rule (422); in
// the path of GET /locations/{id}/stock it is the resource (404).
const (
	locationNotFoundCode    = "location_not_found"
	locationNotFoundMessage = "La sede no existe."
	variantNotFoundCode     = "variant_not_found"
	variantNotFoundMessage  = "La variante no existe."
)

// errorRules maps the inventory domain errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrLocationNameTaken, Status: http.StatusConflict, Code: "location_name_taken", Message: "Ya existe una sede con ese nombre."},
	{Err: domain.ErrInsufficientStock, Status: http.StatusConflict, Code: "insufficient_stock", Message: "No hay stock suficiente para esta operación.", Describe: describeVariantStock},
	{Err: domain.ErrAdjustmentReasonRequired, Status: http.StatusUnprocessableEntity, Code: "adjustment_reason_required", Message: "Escriba el motivo del ajuste."},
	{Err: domain.ErrAdjustmentReasonTooLong, Status: http.StatusUnprocessableEntity, Code: "adjustment_reason_too_long", Message: "El motivo del ajuste puede tener máximo 500 caracteres."},
	{Err: domain.ErrSameLocationTransfer, Status: http.StatusUnprocessableEntity, Code: "same_location_transfer", Message: "La sede de origen y la de destino deben ser distintas."},
	{Err: domain.ErrUnitNotAvailable, Status: http.StatusUnprocessableEntity, Code: "unit_not_available", Message: "El producto no tiene esa unidad."},
	{Err: domain.ErrQuantityPrecision, Status: http.StatusUnprocessableEntity, Code: "quantity_precision", Message: "La cantidad es demasiado pequeña para esa unidad: en la unidad base tendría más de 4 decimales."},
	{Err: domain.ErrInvalidQuantity, Status: http.StatusUnprocessableEntity, Code: "invalid_quantity", Message: "La cantidad no es válida: distinta de cero y con máximo 4 decimales."},
	{Err: domain.ErrInvalidUnitCost, Status: http.StatusUnprocessableEntity, Code: "invalid_unit_cost", Message: "El costo no puede ser negativo y tiene máximo 4 decimales."},
	{Err: domain.ErrInvalidLocationName, Status: http.StatusUnprocessableEntity, Code: "invalid_location_name", Message: "Escriba el nombre de la sede (máximo 100 caracteres)."},
	{Err: domain.ErrVariantNotFound, Status: http.StatusUnprocessableEntity, Code: variantNotFoundCode, Message: variantNotFoundMessage},
	{Err: domain.ErrLocationNotFound, Status: http.StatusUnprocessableEntity, Code: locationNotFoundCode, Message: locationNotFoundMessage},
	{Err: domain.ErrInvalidReorderRange, Status: http.StatusUnprocessableEntity, Code: "invalid_reorder_range", Message: "El stock mínimo no puede ser negativo ni mayor que el máximo, y cada uno tiene máximo 4 decimales."},
	{Err: domain.ErrReorderRuleNotFound, Status: http.StatusNotFound, Code: "reorder_rule_not_found", Message: "La variante no tiene stock mínimo en esta sede."},
	{Err: domain.ErrCountNotFound, Status: http.StatusNotFound, Code: "stock_count_not_found", Message: "El conteo no existe."},
	{Err: domain.ErrCountEntryNotFound, Status: http.StatusNotFound, Code: "count_entry_not_found", Message: "Ese conteo no existe o ya se borró."},
	{Err: domain.ErrCountInProgress, Status: http.StatusConflict, Code: "stock_count_in_progress", Message: "La sede ya tiene un conteo abierto. Apruébelo o cancélelo antes de iniciar otro."},
	{Err: domain.ErrCountNotOpen, Status: http.StatusConflict, Code: "stock_count_not_open", Message: "El conteo ya se aprobó o se canceló."},
	{Err: domain.ErrCountNeedsReview, Status: http.StatusConflict, Code: "stock_count_needs_review", Message: "Hay productos en revisión: pida recontarlos antes de aprobar."},
	{Err: domain.ErrCountEntryNotYours, Status: http.StatusForbidden, Code: "count_entry_not_yours", Message: "Solo puede borrar lo que usted contó."},
	{Err: domain.ErrVariantNotInCount, Status: http.StatusUnprocessableEntity, Code: "variant_not_in_count", Message: "Ese producto no está en este conteo."},
	{Err: domain.ErrCategoryNotFound, Status: http.StatusUnprocessableEntity, Code: "category_not_found", Message: "Una de las categorías no existe."},
	{Err: domain.ErrInvalidCountMode, Status: http.StatusUnprocessableEntity, Code: "invalid_count_mode", Message: "El modo del conteo no es válido."},
	{Err: domain.ErrLocationInactive, Status: http.StatusUnprocessableEntity, Code: "location_inactive", Message: "La sede está inactiva. Actívela antes de mover mercancía en ella."},
}

// describeVariantStock names the variant without stock when the error says
// which one it is (an adjustment of a stock count).
func describeVariantStock(err error) (string, map[string]string, bool) {
	var stockErr *app.VariantStockError
	if !errors.As(err, &stockErr) {
		return "", nil, false
	}
	return "Un ajuste dejaría sin stock un producto que se vendió después de contarlo. Pida recontarlo.",
		map[string]string{"variant_id": stockErr.VariantID.String()}, true
}
