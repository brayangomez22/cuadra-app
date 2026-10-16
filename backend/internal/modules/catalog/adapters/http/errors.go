package httpadapter

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// unauthorizedMessage answers a request that reaches the module without a
// principal, which RequireAuth should have stopped.
const unauthorizedMessage = "Su sesión no es válida o expiró. Inicie sesión de nuevo."

// errorRules maps the catalog domain errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrProductNotFound, Status: http.StatusNotFound, Code: "product_not_found", Message: "El producto no existe."},
	{Err: domain.ErrVariantNotFound, Status: http.StatusNotFound, Code: "variant_not_found", Message: "La variante no existe en este producto."},
	{Err: domain.ErrSKUTaken, Status: http.StatusConflict, Code: "sku_taken", Message: "Ya existe un producto o una variante con ese SKU."},
	{Err: domain.ErrBarcodeTaken, Status: http.StatusConflict, Code: "barcode_taken", Message: "Ya existe un producto o una variante con ese código de barras."},
	{Err: domain.ErrCategoryInUse, Status: http.StatusConflict, Code: "category_in_use", Message: "La categoría tiene productos o subcategorías. Muévelos antes de borrarla."},
	{Err: domain.ErrProductInactive, Status: http.StatusUnprocessableEntity, Code: "product_inactive", Message: "El producto está inactivo. Actívelo antes de cambiarle el precio."},
	{Err: domain.ErrCategoryCycle, Status: http.StatusUnprocessableEntity, Code: "category_cycle", Message: "Una categoría no puede quedar dentro de sus propias subcategorías."},
	{Err: domain.ErrInvalidCategoryParent, Status: http.StatusUnprocessableEntity, Code: "invalid_category_parent", Message: "Una categoría no puede ser su propia categoría padre."},
	{Err: domain.ErrInvalidSKU, Status: http.StatusUnprocessableEntity, Code: "invalid_sku", Message: "El SKU no es válido: sin espacios y máximo 64 caracteres."},
	{Err: domain.ErrInvalidBarcode, Status: http.StatusUnprocessableEntity, Code: "invalid_barcode", Message: "El código de barras no es válido: sin espacios y máximo 64 caracteres."},
	{Err: domain.ErrInvalidProductName, Status: http.StatusUnprocessableEntity, Code: "invalid_product_name", Message: "Escriba el nombre del producto (máximo 200 caracteres)."},
	{Err: domain.ErrInvalidDescription, Status: http.StatusUnprocessableEntity, Code: "invalid_description", Message: "La descripción puede tener máximo 2.000 caracteres."},
	{Err: domain.ErrInvalidAxisValues, Status: http.StatusUnprocessableEntity, Code: "invalid_axis_values", Message: "Los ejes de la variante no son válidos: máximo 5, con nombre y valor (máximo 50 y 100 caracteres)."},
	{Err: domain.ErrVariantAxesMismatch, Status: http.StatusUnprocessableEntity, Code: "variant_axes_mismatch", Message: "La variante debe usar los mismos ejes que las demás variantes del producto."},
	{Err: domain.ErrDuplicateVariant, Status: http.StatusUnprocessableEntity, Code: "duplicate_variant", Message: "Ya existe una variante del producto con esos valores."},
	{Err: domain.ErrInvalidUnit, Status: http.StatusUnprocessableEntity, Code: "invalid_unit", Message: "La unidad de medida no es válida."},
	{Err: domain.ErrUnitRequired, Status: http.StatusUnprocessableEntity, Code: "unit_required", Message: "Indique la unidad de medida (base_unit_id) o su código (base_unit), solo uno de los dos."},
	{Err: domain.ErrInvalidUnitCode, Status: http.StatusUnprocessableEntity, Code: "invalid_unit_code", Message: "El código de la unidad no es válido: sin espacios y máximo 16 caracteres."},
	{Err: domain.ErrInvalidUnitName, Status: http.StatusUnprocessableEntity, Code: "invalid_unit_name", Message: "Escriba el nombre de la unidad (máximo 50 caracteres)."},
	{Err: domain.ErrInvalidPrice, Status: http.StatusUnprocessableEntity, Code: "invalid_price", Message: "El precio debe ser mayor que cero, con máximo 4 decimales."},
	{Err: domain.ErrInvalidCost, Status: http.StatusUnprocessableEntity, Code: "invalid_cost", Message: "El costo no puede ser negativo y tiene máximo 4 decimales."},
	{Err: domain.ErrInvalidTaxRate, Status: http.StatusUnprocessableEntity, Code: "invalid_tax_rate", Message: "La tasa de IVA debe estar entre 0 y 1 (por ejemplo, 0.19)."},
	{Err: domain.ErrInvalidCategoryName, Status: http.StatusUnprocessableEntity, Code: "invalid_category_name", Message: "Escriba el nombre de la categoría (máximo 100 caracteres)."},
	{Err: domain.ErrCategoryNotFound, Status: http.StatusUnprocessableEntity, Code: "category_not_found", Message: "La categoría no existe."},
	// Before ErrUnitNotFound, which it wraps.
	{Err: errUnitPathNotFound, Status: http.StatusNotFound, Code: "unit_not_found", Message: "La unidad de medida no existe."},
	{Err: domain.ErrUnitNotFound, Status: http.StatusUnprocessableEntity, Code: "unit_not_found", Message: "La unidad de medida no existe."},
	{Err: domain.ErrUnitCodeTaken, Status: http.StatusConflict, Code: "unit_code_taken", Message: "Ya existe una unidad de medida con ese código."},
	{Err: domain.ErrUnitInUse, Status: http.StatusConflict, Code: "unit_in_use", Message: "Hay productos que usan esta unidad. Cámbieles la unidad antes de borrarla."},
}
