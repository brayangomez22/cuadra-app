package httpadapter

import (
	"errors"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// unauthorizedMessage answers a request that reaches the module without a
// principal, which RequireAuth should have stopped.
const unauthorizedMessage = "Su sesión no es válida o expiró. Inicie sesión de nuevo."

// errorRules maps the sales domain errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{
		Err: domain.ErrInsufficientStock, Status: http.StatusConflict, Code: "insufficient_stock",
		Message:  "No hay stock suficiente para uno de los productos.",
		Describe: describeInsufficientStock,
	},
	{Err: domain.ErrSaleNotConfirmed, Status: http.StatusConflict, Code: "sale_not_confirmed", Message: "La venta no está confirmada; puede que ya esté anulada."},
	{Err: domain.ErrSaleNotFound, Status: http.StatusNotFound, Code: "sale_not_found", Message: "La venta no existe."},
	{Err: domain.ErrDiscountExceedsLimit, Status: http.StatusForbidden, Code: "discount_exceeds_limit", Message: "El descuento supera el máximo que puede dar."},
	{Err: domain.ErrVariantRequired, Status: http.StatusUnprocessableEntity, Code: "variant_required", Message: "Indique la variante (variant_id) o el producto (product_id) de cada línea, solo uno de los dos."},
	{Err: domain.ErrProductNotFound, Status: http.StatusUnprocessableEntity, Code: "product_not_found", Message: "Uno de los productos no existe."},
	{Err: domain.ErrProductInactive, Status: http.StatusUnprocessableEntity, Code: "product_inactive", Message: "Uno de los productos está inactivo y no se puede vender."},
	{Err: domain.ErrLocationNotFound, Status: http.StatusUnprocessableEntity, Code: "location_not_found", Message: "La sede no existe."},
	{Err: domain.ErrLocationInactive, Status: http.StatusUnprocessableEntity, Code: "location_inactive", Message: "La sede está inactiva. Actívela antes de vender en ella."},
	{Err: domain.ErrInsufficientPayment, Status: http.StatusUnprocessableEntity, Code: "insufficient_payment", Message: "Los pagos no alcanzan para cubrir el total."},
	{Err: domain.ErrNonCashOverpayment, Status: http.StatusUnprocessableEntity, Code: "non_cash_overpayment", Message: "Solo el efectivo da cambio: los demás pagos no pueden superar el total."},
	{Err: domain.ErrSaleWithoutLines, Status: http.StatusUnprocessableEntity, Code: "sale_without_lines", Message: "Agregue al menos un producto a la venta."},
	{Err: domain.ErrInvalidQuantity, Status: http.StatusUnprocessableEntity, Code: "invalid_quantity", Message: "La cantidad debe ser mayor que cero y tener máximo 4 decimales."},
	{Err: domain.ErrInvalidDiscount, Status: http.StatusUnprocessableEntity, Code: "invalid_discount", Message: "El descuento no puede ser negativo, tiene máximo 2 decimales y no puede superar el valor de la línea."},
	{Err: domain.ErrInvalidPaymentAmount, Status: http.StatusUnprocessableEntity, Code: "invalid_payment_amount", Message: "Cada pago debe ser mayor que cero y tener máximo 2 decimales."},
	{Err: domain.ErrInvalidPaymentMethod, Status: http.StatusUnprocessableEntity, Code: "invalid_payment_method", Message: "El medio de pago no es válido."},
	{Err: domain.ErrInvalidVoidReason, Status: http.StatusUnprocessableEntity, Code: "invalid_void_reason", Message: "Escriba el motivo de la anulación (máximo 500 caracteres)."},

	// Cash sessions.
	{Err: domain.ErrCashSessionRequired, Status: http.StatusConflict, Code: "cash_session_required", Message: "Se necesita una caja abierta para vender."},
	{Err: domain.ErrCashSessionClosed, Status: http.StatusConflict, Code: "cash_session_closed", Message: "La caja ya está cerrada. Para seguir vendiendo, abra una nueva."},
	{Err: domain.ErrCashSessionAlreadyOpen, Status: http.StatusConflict, Code: "cash_session_already_open", Message: "Ya tiene una caja abierta. Ciérrela antes de abrir otra."},
	{Err: domain.ErrCashSessionNotOwner, Status: http.StatusForbidden, Code: "cash_session_not_owner", Message: "Esta caja es de otro usuario: solo quien la abrió registra movimientos en ella o la cierra."},
	{Err: domain.ErrCashSessionNotFound, Status: http.StatusNotFound, Code: "cash_session_not_found", Message: "La caja no existe."},
	{Err: domain.ErrCashSessionLocationMismatch, Status: http.StatusUnprocessableEntity, Code: "cash_session_location_mismatch", Message: "La caja está abierta en otra sede. Venda en esa sede o cierre la caja."},
	{Err: domain.ErrInvalidOpeningFloat, Status: http.StatusUnprocessableEntity, Code: "invalid_opening_float", Message: "La base no puede ser negativa y tiene máximo 2 decimales."},
	{Err: domain.ErrInvalidCashMovementKind, Status: http.StatusUnprocessableEntity, Code: "invalid_cash_movement_kind", Message: "El movimiento debe ser un ingreso o un egreso."},
	{Err: domain.ErrInvalidCashMovementAmount, Status: http.StatusUnprocessableEntity, Code: "invalid_cash_movement_amount", Message: "El monto debe ser mayor que cero y tener máximo 2 decimales."},
	{Err: domain.ErrInvalidCashMovementReason, Status: http.StatusUnprocessableEntity, Code: "invalid_cash_movement_reason", Message: "Escriba el motivo del movimiento (máximo 200 caracteres)."},
	{Err: domain.ErrInvalidDenomination, Status: http.StatusUnprocessableEntity, Code: "invalid_denomination", Message: "Una de las denominaciones no es un billete o moneda vigente."},
	{Err: domain.ErrDuplicateDenomination, Status: http.StatusUnprocessableEntity, Code: "duplicate_denomination", Message: "Cada denominación va una sola vez en el conteo."},
	{Err: domain.ErrInvalidDenominationQuantity, Status: http.StatusUnprocessableEntity, Code: "invalid_denomination_quantity", Message: "La cantidad de billetes o monedas no es válida."},
	{Err: domain.ErrInvalidCashNotes, Status: http.StatusUnprocessableEntity, Code: "invalid_cash_notes", Message: "Las notas del cierre tienen máximo 500 caracteres."},
}

// describeInsufficientStock names the variant without stock, so the cashier
// knows which line to change: the product's name and the variant's label,
// the catalog's, not personal data. details.product_id is deprecated.
func describeInsufficientStock(err error) (string, map[string]string, bool) {
	var e *domain.InsufficientStockError
	if !errors.As(err, &e) || e.ProductName == "" {
		return "", nil, false
	}
	return "No hay stock suficiente de " + e.Description() + ".",
		map[string]string{"variant_id": e.VariantID.String(), "product_id": e.ProductID.String()}, true
}
