package httpadapter

import (
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// errorRules maps the receivables domain errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{Err: domain.ErrCustomerNotFound, Status: http.StatusNotFound, Code: "customer_not_found", Message: "El cliente no existe."},
	{Err: domain.ErrCashSessionRequired, Status: http.StatusConflict, Code: "cash_session_required", Message: "Abra la caja para recibir abonos en efectivo."},
	{Err: domain.ErrInvalidCreditLimit, Status: http.StatusUnprocessableEntity, Code: "invalid_credit_limit", Message: "El cupo debe ser un monto en pesos, de cero en adelante."},
	{Err: domain.ErrInvalidCreditTerm, Status: http.StatusUnprocessableEntity, Code: "invalid_credit_term", Message: "El plazo debe estar entre 0 y 365 días."},
	{Err: domain.ErrInvalidPaymentMethod, Status: http.StatusUnprocessableEntity, Code: "invalid_payment_method", Message: "El medio de pago del abono no es válido."},
	{Err: domain.ErrInvalidPaymentAmount, Status: http.StatusUnprocessableEntity, Code: "invalid_payment_amount", Message: "El abono debe ser un monto en pesos mayor que cero."},
	{Err: domain.ErrPaymentExceedsBalance, Status: http.StatusUnprocessableEntity, Code: "payment_exceeds_balance", Message: "El abono es mayor que lo que debe el cliente."},
}
