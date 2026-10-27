package httpadapter

import (
	"fmt"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/imports/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// errorRules maps the imports errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrImportNotFound, Status: http.StatusNotFound, Code: "import_not_found", Message: "La importación no existe."},
	{Err: domain.ErrImportNotValidated, Status: http.StatusConflict, Code: "import_not_validated", Message: "Solo se puede confirmar una importación validada y sin errores."},
	{Err: domain.ErrFileTooLarge, Status: http.StatusRequestEntityTooLarge, Code: "file_too_large", Message: "El archivo pasa de 10 MB. Divídalo en varios archivos."},
	{Err: domain.ErrUnreadableFile, Status: http.StatusUnprocessableEntity, Code: "unreadable_file", Message: "No se pudo leer el archivo. Guárdelo de nuevo como .xlsx o .csv e intente otra vez."},
	{Err: errUnsupportedFormat, Status: http.StatusUnprocessableEntity, Code: "unsupported_format", Message: "El archivo debe ser .xlsx o .csv."},
	{Err: app.ErrLocationNotFound, Status: http.StatusUnprocessableEntity, Code: "location_not_found", Message: "La sede no existe o está inactiva."},
	{Err: domain.ErrInvalidImport, Status: http.StatusUnprocessableEntity, Code: "invalid_import", Message: "Los datos de la importación no son válidos."},
	{Err: errInvalidForm, Status: http.StatusBadRequest, Code: "invalid_request", Message: httpx.InvalidRequestMessage},
}

// rowProblems are the user's words for the problems the import finds on its
// own; the catalog's come with their message.
var rowProblems = map[string]string{
	domain.CodeMissingColumn:    "falta esta columna, que es obligatoria",
	domain.CodeUnknownColumn:    "la columna no es de la plantilla; revise el nombre o descargue la plantilla de nuevo",
	domain.CodeDuplicateColumn:  "la columna está repetida",
	domain.CodeEmptyFile:        "el archivo no tiene filas de productos",
	domain.CodeTooManyRows:      "el archivo pasa de 20.000 filas; divídalo en varios archivos",
	domain.CodeRequired:         "falta el valor, que es obligatorio",
	domain.CodeNotANumber:       "no es un número",
	domain.CodeNegativeNumber:   "no puede ser negativo",
	domain.CodeInvalidBoolean:   "escriba sí o no",
	domain.CodeDuplicateSKU:     "el SKU ya está en otra fila del archivo",
	domain.CodeProductSplit:     "el producto ya apareció más arriba; sus filas deben ir seguidas",
	domain.CodeVariantsInactive: "el producto ya está en la fila anterior; para varias variantes active el módulo de variantes",
}

// rowMessage describes a row error to the user: "Fila 154, columna
// «precio»: no es un número."
func rowMessage(e domain.RowError) string {
	problem := e.Message
	if problem == "" {
		problem = rowProblems[e.Code] + "."
	}
	if e.Column == "" {
		return fmt.Sprintf("Fila %d: %s", e.Row, problem)
	}
	return fmt.Sprintf("Fila %d, columna «%s»: %s", e.Row, e.Column, problem)
}
