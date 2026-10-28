package httpadapter

import (
	"errors"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// errorRules maps the customers domain errors to HTTP. The segment of the
// path goes before the segment of the body: both wrap ErrSegmentNotFound.
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrCustomerNotFound, Status: http.StatusNotFound, Code: "customer_not_found", Message: "El cliente no existe."},
	{Err: errSegmentPathNotFound, Status: http.StatusNotFound, Code: "customer_segment_not_found", Message: "El segmento no existe."},
	{Err: domain.ErrDocumentTaken, Status: http.StatusConflict, Code: "customer_document_taken",
		Message: "Ya existe un cliente con ese documento.", Describe: describeDocument},
	{Err: domain.ErrSegmentNameTaken, Status: http.StatusConflict, Code: "customer_segment_name_taken", Message: "Ya existe un segmento con ese nombre."},
	{Err: domain.ErrSegmentInUse, Status: http.StatusConflict, Code: "customer_segment_in_use", Message: "El segmento tiene clientes. Cámbielos de segmento antes de borrarlo."},
	{Err: domain.ErrSegmentNotFound, Status: http.StatusUnprocessableEntity, Code: "customer_segment_not_found", Message: "El segmento elegido no existe."},
	{Err: domain.ErrInvalidNITCheckDigit, Status: http.StatusUnprocessableEntity, Code: "invalid_nit_check_digit", Message: "El dígito de verificación del NIT no coincide. Revise el número."},
	{Err: domain.ErrInvalidDocument, Status: http.StatusUnprocessableEntity, Code: "invalid_document", Message: "El número de documento no es válido: las cédulas llevan solo dígitos, el NIT su dígito de verificación y el pasaporte letras y números."},
	{Err: domain.ErrInvalidDocumentType, Status: http.StatusUnprocessableEntity, Code: "invalid_document", Message: "El tipo de documento no es válido."},
	{Err: domain.ErrInvalidPersonType, Status: http.StatusUnprocessableEntity, Code: "invalid_person_type", Message: "Indique si el cliente es persona natural o jurídica."},
	{Err: domain.ErrLegalPersonRequiresNIT, Status: http.StatusUnprocessableEntity, Code: "legal_person_requires_nit", Message: "Una persona jurídica se identifica con NIT."},
	{Err: domain.ErrInvalidCustomerName, Status: http.StatusUnprocessableEntity, Code: "invalid_customer_name", Message: "Escriba el nombre del cliente (máximo 200 caracteres)."},
	{Err: domain.ErrInvalidContact, Status: http.StatusUnprocessableEntity, Code: "invalid_customer_contact", Message: "Revise el correo y el teléfono del cliente."},
	{Err: domain.ErrInvalidTaxResponsibilities, Status: http.StatusUnprocessableEntity, Code: "invalid_tax_responsibilities", Message: "Las responsabilidades fiscales no son válidas: sin repetir, y «No responsable» (R-99-PN) va sola."},
	{Err: domain.ErrInvalidSegmentName, Status: http.StatusUnprocessableEntity, Code: "invalid_customer_segment_name", Message: "Escriba el nombre del segmento (máximo 60 caracteres)."},
	{Err: app.ErrInvalidStatus, Status: http.StatusBadRequest, Code: "invalid_request", Message: "El estado del filtro no es válido."},
}

// describeDocument names the document already in use.
func describeDocument(err error) (string, map[string]string, bool) {
	var taken *domain.DocumentTakenError
	if !errors.As(err, &taken) {
		return "", nil, false
	}
	return "Ya existe un cliente con el documento " + taken.Document.String() + ".", nil, true
}
