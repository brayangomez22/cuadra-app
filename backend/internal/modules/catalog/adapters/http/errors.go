package httpadapter

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/catalog/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// errorRules maps the catalog domain errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrProductNotFound, Status: http.StatusNotFound, Code: "product_not_found", Message: "El producto no existe."},
	{Err: domain.ErrVariantNotFound, Status: http.StatusNotFound, Code: "variant_not_found", Message: "La variante no existe en este producto."},
	{Err: domain.ErrSKUTaken, Status: http.StatusConflict, Code: "sku_taken", Message: "Ya existe un producto o una variante con ese SKU."},
	{Err: domain.ErrBarcodeTaken, Status: http.StatusConflict, Code: "barcode_taken", Message: "Ya existe un producto, una variante o una unidad con ese código de barras."},
	{Err: domain.ErrCategoryInUse, Status: http.StatusConflict, Code: "category_in_use", Message: "La categoría tiene productos o subcategorías. Muévelos antes de borrarla."},
	{Err: domain.ErrBaseUnitLocked, Status: http.StatusConflict, Code: "base_unit_locked", Message: "El producto ya tiene movimientos de stock o unidades alternativas contados en su unidad base: no se puede cambiar."},
	{Err: domain.ErrNoVariants, Status: http.StatusUnprocessableEntity, Code: "no_variants", Message: "Agregue al menos una variante al producto."},
	{Err: domain.ErrVariantsOfSeveralProducts, Status: http.StatusUnprocessableEntity, Code: "variants_of_several_products", Message: "Los SKU del producto ya pertenecen a productos distintos del catálogo."},
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
	{Err: domain.ErrInvalidUnitFactor, Status: http.StatusUnprocessableEntity, Code: "invalid_unit_factor", Message: "El factor de la unidad debe ser mayor que cero, con máximo 4 decimales."},
	{Err: domain.ErrAlternateUnitIsBase, Status: http.StatusUnprocessableEntity, Code: "alternate_unit_is_base", Message: "La unidad base del producto no puede ser también una unidad alternativa."},
	{Err: domain.ErrDuplicateVariantUnit, Status: http.StatusUnprocessableEntity, Code: "duplicate_variant_unit", Message: "La variante ya tiene esa unidad."},
	{Err: domain.ErrTooManyVariantUnits, Status: http.StatusUnprocessableEntity, Code: "too_many_variant_units", Message: "Una variante puede tener máximo 10 unidades alternativas."},
	{Err: domain.ErrVariantUnitNotFound, Status: http.StatusNotFound, Code: "variant_unit_not_found", Message: "La variante no tiene esa unidad."},
	{Err: domain.ErrUnitInUse, Status: http.StatusConflict, Code: "unit_in_use", Message: "Hay productos que usan esta unidad, como unidad base o alternativa. Cámbiela en ellos antes de borrarla."},
	{Err: domain.ErrUnknownAttribute, Status: http.StatusUnprocessableEntity, Code: "unknown_attribute",
		Message: "Uno de los atributos no está definido en el catálogo.",
		Describe: describeAttribute(func(name string) string {
			return "El atributo «" + name + "» no está definido en el catálogo."
		})},
	{Err: domain.ErrInvalidAttributeValue, Status: http.StatusUnprocessableEntity, Code: "invalid_attribute_value",
		Message: "El valor de uno de los atributos no es válido.",
		Describe: describeAttribute(func(name string) string {
			return "El valor de «" + name + "» no es válido para ese atributo."
		})},
	{Err: domain.ErrAttributeRequired, Status: http.StatusUnprocessableEntity, Code: "attribute_required",
		Message: "Falta un atributo obligatorio.",
		Describe: describeAttribute(func(name string) string {
			return "Indique «" + name + "»: es obligatorio."
		})},
	{Err: domain.ErrInvalidAttributeKey, Status: http.StatusUnprocessableEntity, Code: "invalid_attribute_key", Message: "La clave del atributo no es válida: minúsculas sin tildes, números y guion bajo, empezando por una letra (máximo 50)."},
	{Err: domain.ErrInvalidAttributeLabel, Status: http.StatusUnprocessableEntity, Code: "invalid_attribute_label", Message: "Escriba la etiqueta del atributo (máximo 100 caracteres)."},
	{Err: domain.ErrInvalidAttributeType, Status: http.StatusUnprocessableEntity, Code: "invalid_attribute_type", Message: "El tipo del atributo no es válido."},
	{Err: domain.ErrInvalidAttributeOptions, Status: http.StatusUnprocessableEntity, Code: "invalid_attribute_options", Message: "Un atributo de lista necesita al menos una opción, sin repetir y de máximo 100 caracteres; los demás tipos no llevan opciones."},
	{Err: domain.ErrAttributeNotFound, Status: http.StatusNotFound, Code: "attribute_not_found", Message: "El atributo no existe."},
	{Err: domain.ErrAttributeKeyTaken, Status: http.StatusConflict, Code: "attribute_key_taken", Message: "Ya existe un atributo con esa clave."},
	{Err: domain.ErrAttributeInUse, Status: http.StatusConflict, Code: "attribute_in_use", Message: "Hay productos con este atributo. Quíteselo antes de borrarlo."},
	{Err: domain.ErrAttributeOptionInUse, Status: http.StatusConflict, Code: "attribute_option_in_use", Message: "Hay productos con una de las opciones que quitó. Cámbieles el valor antes de quitarla."},
	{Err: domain.ErrUnknownAxis, Status: http.StatusUnprocessableEntity, Code: "unknown_axis",
		Message: "Uno de los ejes de la variante no está definido en el catálogo.",
		Describe: describeAxis(func(axis string) string {
			return "El eje «" + axis + "» no está definido en el catálogo. Créelo antes de usarlo."
		})},
	{Err: domain.ErrAxisValueNotAllowed, Status: http.StatusUnprocessableEntity, Code: "axis_value_not_allowed",
		Message: "Uno de los ejes no admite el valor de la variante.",
		Describe: describeAxis(func(axis string) string {
			return "El eje «" + axis + "» no admite ese valor. Agréguelo al eje antes de usarlo."
		})},
	{Err: domain.ErrInvalidAxisName, Status: http.StatusUnprocessableEntity, Code: "invalid_axis_name", Message: "Escriba el nombre del eje (máximo 50 caracteres)."},
	{Err: domain.ErrInvalidAxisAllowedValues, Status: http.StatusUnprocessableEntity, Code: "invalid_axis_allowed_values", Message: "El eje necesita al menos un valor, sin repetir y de máximo 100 caracteres."},
	{Err: domain.ErrAxisNotFound, Status: http.StatusNotFound, Code: "axis_not_found", Message: "El eje no existe."},
	{Err: domain.ErrAxisNameTaken, Status: http.StatusConflict, Code: "axis_name_taken", Message: "Ya existe un eje con ese nombre."},
	{Err: domain.ErrAxisInUse, Status: http.StatusConflict, Code: "axis_in_use", Message: "Hay variantes que usan este eje. Cámbielas antes de borrarlo."},
	{Err: domain.ErrAxisValueInUse, Status: http.StatusConflict, Code: "axis_value_in_use", Message: "Hay variantes con uno de los valores que quitó. Cámbielas antes de quitarlo."},
}

// DescribeError returns the code and the Spanish message the API gives for
// a catalog error; ok is false for an error that is not the catalog's.
func DescribeError(err error) (code, message string, ok bool) {
	_, detail, ok := httpx.Describe(errorRules, err)
	return detail.Code, detail.Message, ok
}
