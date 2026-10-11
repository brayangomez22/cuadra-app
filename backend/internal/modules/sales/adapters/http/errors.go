package httpadapter

import (
	"errors"
	"net/http"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// unauthorizedMessage answers a request that reaches the module without a
// principal, which RequireAuth should have stopped.
const unauthorizedMessage = "Tu sesión no es válida o expiró. Inicia sesión de nuevo."

// errorRules maps the sales domain errors to HTTP.
var errorRules = []httpx.ErrorRule{
	{
		Err: domain.ErrInsufficientStock, Status: http.StatusConflict, Code: "insufficient_stock",
		Message:  "No hay stock suficiente para uno de los productos.",
		Describe: describeInsufficientStock,
	},
	{Err: domain.ErrSaleNotConfirmed, Status: http.StatusConflict, Code: "sale_not_confirmed", Message: "La venta no está confirmada; puede que ya esté anulada."},
	{Err: domain.ErrSaleNotFound, Status: http.StatusNotFound, Code: "sale_not_found", Message: "La venta no existe."},
	{Err: domain.ErrDiscountExceedsLimit, Status: http.StatusForbidden, Code: "discount_exceeds_limit", Message: "El descuento supera el máximo que puedes dar."},
	{Err: domain.ErrProductNotFound, Status: http.StatusUnprocessableEntity, Code: "product_not_found", Message: "Uno de los productos no existe."},
	{Err: domain.ErrProductInactive, Status: http.StatusUnprocessableEntity, Code: "product_inactive", Message: "Uno de los productos está inactivo y no se puede vender."},
	{Err: domain.ErrLocationNotFound, Status: http.StatusUnprocessableEntity, Code: "location_not_found", Message: "La sede no existe."},
	{Err: domain.ErrLocationInactive, Status: http.StatusUnprocessableEntity, Code: "location_inactive", Message: "La sede está inactiva. Actívala antes de vender en ella."},
	{Err: domain.ErrInsufficientPayment, Status: http.StatusUnprocessableEntity, Code: "insufficient_payment", Message: "Los pagos no alcanzan para cubrir el total."},
	{Err: domain.ErrNonCashOverpayment, Status: http.StatusUnprocessableEntity, Code: "non_cash_overpayment", Message: "Solo el efectivo da cambio: los demás pagos no pueden superar el total."},
	{Err: domain.ErrSaleWithoutLines, Status: http.StatusUnprocessableEntity, Code: "sale_without_lines", Message: "Agrega al menos un producto a la venta."},
	{Err: domain.ErrInvalidQuantity, Status: http.StatusUnprocessableEntity, Code: "invalid_quantity", Message: "La cantidad debe ser mayor que cero y tener máximo 4 decimales."},
	{Err: domain.ErrInvalidDiscount, Status: http.StatusUnprocessableEntity, Code: "invalid_discount", Message: "El descuento no puede ser negativo, tiene máximo 2 decimales y no puede superar el valor de la línea."},
	{Err: domain.ErrInvalidPaymentAmount, Status: http.StatusUnprocessableEntity, Code: "invalid_payment_amount", Message: "Cada pago debe ser mayor que cero y tener máximo 2 decimales."},
	{Err: domain.ErrInvalidPaymentMethod, Status: http.StatusUnprocessableEntity, Code: "invalid_payment_method", Message: "El medio de pago no es válido."},
	{Err: domain.ErrInvalidVoidReason, Status: http.StatusUnprocessableEntity, Code: "invalid_void_reason", Message: "Escribe el motivo de la anulación (máximo 500 caracteres)."},
}

// describeInsufficientStock names the product without stock, so the cashier
// knows which line to change. The name is the catalog's, not personal data.
func describeInsufficientStock(err error) (string, map[string]string, bool) {
	var e *domain.InsufficientStockError
	if !errors.As(err, &e) || e.ProductName == "" {
		return "", nil, false
	}
	return "No hay stock suficiente de " + e.ProductName + ".", map[string]string{"product_id": e.ProductID.String()}, true
}
