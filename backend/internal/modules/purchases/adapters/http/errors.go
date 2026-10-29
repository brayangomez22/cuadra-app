package httpadapter

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// errorRules maps the purchases domain errors to HTTP. The errors of a line
// say which in details.line (from 0) and in the message (from 1).
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrPurchaseNotFound, Status: http.StatusNotFound, Code: "purchase_not_found", Message: "La compra no existe."},
	{Err: errSupplierPathNotFound, Status: http.StatusNotFound, Code: "supplier_not_found", Message: "El proveedor no existe."},
	{Err: domain.ErrNITTaken, Status: http.StatusConflict, Code: "supplier_nit_taken", Message: "Ya existe un proveedor con ese NIT."},
	{Err: domain.ErrInvoiceTaken, Status: http.StatusConflict, Code: "purchase_invoice_taken", Message: "Esa factura del proveedor ya está registrada."},
	{Err: domain.ErrInvalidNITCheckDigit, Status: http.StatusUnprocessableEntity, Code: "invalid_nit_check_digit", Message: "El dígito de verificación del NIT no coincide. Revise el número."},
	{Err: domain.ErrInvalidNIT, Status: http.StatusUnprocessableEntity, Code: "invalid_nit", Message: "Escriba el NIT con su dígito de verificación, por ejemplo 900123456-8."},
	{Err: domain.ErrInvalidSupplierName, Status: http.StatusUnprocessableEntity, Code: "invalid_supplier_name", Message: "Escriba el nombre del proveedor (máximo 200 caracteres)."},
	{Err: domain.ErrInvalidContact, Status: http.StatusUnprocessableEntity, Code: "invalid_supplier_contact", Message: "Revise el contacto, el correo y el teléfono del proveedor."},
	{Err: domain.ErrInvalidPaymentTerm, Status: http.StatusUnprocessableEntity, Code: "invalid_payment_term", Message: "El plazo de pago va de 0 a 365 días."},
	{Err: domain.ErrSupplierNotFound, Status: http.StatusUnprocessableEntity, Code: "supplier_not_found", Message: "El proveedor elegido no existe."},
	{Err: domain.ErrSupplierInactive, Status: http.StatusUnprocessableEntity, Code: "supplier_inactive", Message: "El proveedor está inactivo: actívelo para registrarle compras."},
	{Err: domain.ErrLocationNotFound, Status: http.StatusUnprocessableEntity, Code: "location_not_found", Message: "La sede no existe."},
	{Err: domain.ErrLocationInactive, Status: http.StatusUnprocessableEntity, Code: "location_inactive", Message: "La sede está inactiva."},
	{Err: domain.ErrInvalidInvoiceNumber, Status: http.StatusUnprocessableEntity, Code: "invalid_invoice_number", Message: "Escriba el número de la factura (máximo 50 caracteres)."},
	{Err: domain.ErrInvoiceDateInFuture, Status: http.StatusUnprocessableEntity, Code: "invoice_date_in_future", Message: "La fecha de la factura no puede ser posterior a hoy."},
	{Err: domain.ErrDueBeforeIssue, Status: http.StatusUnprocessableEntity, Code: "due_before_issue", Message: "El vencimiento no puede ser anterior a la fecha de la factura."},
	{Err: domain.ErrInvalidNotes, Status: http.StatusUnprocessableEntity, Code: "invalid_purchase_notes", Message: "Las notas tienen máximo 500 caracteres."},
	{Err: domain.ErrInvalidPayment, Status: http.StatusUnprocessableEntity, Code: "invalid_request", Message: "La forma de pago no es válida."},
	{Err: domain.ErrNoLines, Status: http.StatusUnprocessableEntity, Code: "invalid_purchase_lines", Message: "Agregue al menos un producto a la compra."},
	{Err: domain.ErrTooManyLines, Status: http.StatusUnprocessableEntity, Code: "invalid_purchase_lines", Message: "Una compra tiene máximo 200 líneas: regístrela en partes."},
	lineRule(domain.ErrVariantNotFound, "variant_not_found", "El producto no existe."),
	lineRule(domain.ErrVariantInactive, "variant_inactive", "El producto está inactivo."),
	lineRule(domain.ErrUnitNotAvailable, "unit_not_available", "El producto no tiene esa unidad de compra."),
	lineRule(domain.ErrQuantityPrecision, "quantity_precision", "La cantidad en la unidad base necesita más de 4 decimales."),
	lineRule(domain.ErrInvalidQuantity, "invalid_quantity", "La cantidad debe ser mayor que cero, con hasta 4 decimales."),
	lineRule(domain.ErrInvalidUnitCost, "invalid_unit_cost", "El costo no puede ser negativo."),
	lineRule(domain.ErrInvalidTaxRate, "invalid_tax_rate", "La tasa de IVA debe estar entre 0 y 1."),
	{Err: app.ErrInvalidStatus, Status: http.StatusBadRequest, Code: "invalid_request", Message: "El estado del filtro no es válido."},
}

// errSupplierPathNotFound is the supplier of the path that does not exist:
// a 404, unlike the supplier of a purchase's body.
var errSupplierPathNotFound = errors.New("purchases: supplier of the path not found")

// lineRule is the 422 of an error of one line; with an *app.LineError it
// names the line.
func lineRule(err error, code, message string) httpx.ErrorRule {
	return httpx.ErrorRule{
		Err: err, Status: http.StatusUnprocessableEntity, Code: code, Message: message,
		Describe: func(err error) (string, map[string]string, bool) {
			var line *app.LineError
			if !errors.As(err, &line) {
				return "", nil, false
			}
			return "Línea " + strconv.Itoa(line.Index+1) + ": " + message,
				map[string]string{"line": strconv.Itoa(line.Index)}, true
		},
	}
}
