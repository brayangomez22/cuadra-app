package httpadapter

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// unauthorizedMessage answers a request that reaches the module without a
// principal, which RequireAuth should have stopped.
const unauthorizedMessage = "Tu sesión no es válida o expiró. Inicia sesión de nuevo."

// errorRules maps the catalog domain errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrProductNotFound, Status: http.StatusNotFound, Code: "product_not_found", Message: "El producto no existe."},
	{Err: domain.ErrSKUTaken, Status: http.StatusConflict, Code: "sku_taken", Message: "Ya existe un producto con ese SKU."},
	{Err: domain.ErrBarcodeTaken, Status: http.StatusConflict, Code: "barcode_taken", Message: "Ya existe un producto con ese código de barras."},
	{Err: domain.ErrCategoryInUse, Status: http.StatusConflict, Code: "category_in_use", Message: "La categoría tiene productos o subcategorías. Muévelos antes de borrarla."},
	{Err: domain.ErrProductInactive, Status: http.StatusUnprocessableEntity, Code: "product_inactive", Message: "El producto está inactivo. Actívalo antes de cambiarle el precio."},
	{Err: domain.ErrCategoryCycle, Status: http.StatusUnprocessableEntity, Code: "category_cycle", Message: "Una categoría no puede quedar dentro de sus propias subcategorías."},
	{Err: domain.ErrInvalidCategoryParent, Status: http.StatusUnprocessableEntity, Code: "invalid_category_parent", Message: "Una categoría no puede ser su propia categoría padre."},
	{Err: domain.ErrInvalidSKU, Status: http.StatusUnprocessableEntity, Code: "invalid_sku", Message: "El SKU no es válido: sin espacios y máximo 64 caracteres."},
	{Err: domain.ErrInvalidBarcode, Status: http.StatusUnprocessableEntity, Code: "invalid_barcode", Message: "El código de barras no es válido: sin espacios y máximo 64 caracteres."},
	{Err: domain.ErrInvalidProductName, Status: http.StatusUnprocessableEntity, Code: "invalid_product_name", Message: "Escribe el nombre del producto (máximo 200 caracteres)."},
	{Err: domain.ErrInvalidDescription, Status: http.StatusUnprocessableEntity, Code: "invalid_description", Message: "La descripción puede tener máximo 2.000 caracteres."},
	{Err: domain.ErrInvalidUnit, Status: http.StatusUnprocessableEntity, Code: "invalid_unit", Message: "La unidad de medida no es válida."},
	{Err: domain.ErrInvalidPrice, Status: http.StatusUnprocessableEntity, Code: "invalid_price", Message: "El precio debe ser mayor que cero, con máximo 4 decimales."},
	{Err: domain.ErrInvalidCost, Status: http.StatusUnprocessableEntity, Code: "invalid_cost", Message: "El costo no puede ser negativo y tiene máximo 4 decimales."},
	{Err: domain.ErrInvalidTaxRate, Status: http.StatusUnprocessableEntity, Code: "invalid_tax_rate", Message: "La tasa de IVA debe estar entre 0 y 1 (por ejemplo, 0.19)."},
	{Err: domain.ErrInvalidCategoryName, Status: http.StatusUnprocessableEntity, Code: "invalid_category_name", Message: "Escribe el nombre de la categoría (máximo 100 caracteres)."},
	{Err: domain.ErrCategoryNotFound, Status: http.StatusUnprocessableEntity, Code: "category_not_found", Message: "La categoría no existe."},
}
