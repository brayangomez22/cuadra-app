package domain

import (
	"errors"

	"github.com/google/uuid"
)

// Domain errors of the sales module. The HTTP adapter maps them to status codes.
var (
	ErrInvalidSaleID        = errors.New("sales: invalid sale id")
	ErrInvalidTenantID      = errors.New("sales: invalid tenant id")
	ErrInvalidLocationID    = errors.New("sales: invalid location id")
	ErrInvalidUserID        = errors.New("sales: invalid user id")
	ErrInvalidCustomerID    = errors.New("sales: invalid customer id")
	ErrInvalidProductID     = errors.New("sales: invalid product id")
	ErrInvalidProductName   = errors.New("sales: invalid product name")
	ErrInvalidVariantID     = errors.New("sales: invalid variant id")
	ErrInvalidVariantLabel  = errors.New("sales: invalid variant label")
	ErrInvalidPriceRule     = errors.New("sales: invalid price rule")
	ErrInvalidQuantity      = errors.New("sales: invalid quantity")
	ErrInvalidUnitPrice     = errors.New("sales: invalid unit price")
	ErrInvalidTaxRate       = errors.New("sales: invalid tax rate")
	ErrInvalidDiscount      = errors.New("sales: invalid discount")
	ErrInvalidDiscountLimit = errors.New("sales: invalid discount limit")
	ErrInvalidPaymentMethod = errors.New("sales: invalid payment method")
	ErrInvalidPaymentAmount = errors.New("sales: invalid payment amount")
	ErrInvalidSaleNumber    = errors.New("sales: invalid sale number")
	ErrInvalidVoidReason    = errors.New("sales: invalid void reason")

	ErrInvalidCashSessionID        = errors.New("sales: invalid cash session id")
	ErrInvalidOpeningFloat         = errors.New("sales: invalid opening float")
	ErrInvalidCashMovementID       = errors.New("sales: invalid cash movement id")
	ErrInvalidCashMovementKind     = errors.New("sales: invalid cash movement kind")
	ErrInvalidCashMovementAmount   = errors.New("sales: invalid cash movement amount")
	ErrInvalidCashMovementReason   = errors.New("sales: invalid cash movement reason")
	ErrInvalidDenomination         = errors.New("sales: invalid denomination")
	ErrInvalidDenominationQuantity = errors.New("sales: invalid denomination quantity")
	ErrDuplicateDenomination       = errors.New("sales: duplicate denomination")
	ErrInvalidCashNotes            = errors.New("sales: invalid cash closing notes")
	ErrInvalidCashActivity         = errors.New("sales: invalid cash activity")

	ErrDiscountExceedsLimit = errors.New("sales: discount exceeds the user's limit")
	ErrSaleWithoutLines     = errors.New("sales: sale without lines")
	ErrInsufficientPayment  = errors.New("sales: payments do not cover the total")
	ErrNonCashOverpayment   = errors.New("sales: non-cash payments exceed the total")
	ErrSaleNotDraft         = errors.New("sales: sale is not a draft")
	ErrSaleNotConfirmed     = errors.New("sales: sale is not confirmed")
	ErrSaleNumberAssigned   = errors.New("sales: sale number already assigned")
	ErrInconsistentSale     = errors.New("sales: stored sale is inconsistent")

	ErrCashSessionRequired         = errors.New("sales: no open cash session")
	ErrCashSessionAlreadyOpen      = errors.New("sales: the user already has an open cash session")
	ErrCashSessionClosed           = errors.New("sales: cash session is closed")
	ErrCashSessionNotOwner         = errors.New("sales: cash session belongs to another user")
	ErrCashSessionLocationMismatch = errors.New("sales: sale location differs from the cash session's")
	ErrInconsistentCashSession     = errors.New("sales: stored cash session is inconsistent")
	ErrCashSessionNotFound         = errors.New("sales: cash session not found")

	ErrSaleNotFound      = errors.New("sales: sale not found")
	ErrProductNotFound   = errors.New("sales: product not found")
	ErrProductInactive   = errors.New("sales: product is inactive")
	ErrLocationNotFound  = errors.New("sales: location not found")
	ErrLocationInactive  = errors.New("sales: location is inactive")
	ErrCustomerNotFound  = errors.New("sales: customer not found")
	ErrCustomerInactive  = errors.New("sales: customer is inactive")
	ErrInsufficientStock = errors.New("sales: insufficient stock")
	ErrQuantityPrecision = errors.New("sales: quantity in the base unit needs more than four decimals")
	ErrInvalidUnitFactor = errors.New("sales: invalid unit factor")
	ErrInvalidUnitCode   = errors.New("sales: invalid unit code")
	ErrUnitNotAvailable  = errors.New("sales: the variant is not sold in that unit")
)

// InsufficientStockError is ErrInsufficientStock for one variant of the
// sale: errors.Is(err, ErrInsufficientStock) holds. The StockReserver fills
// VariantID; ConfirmSale adds its product, name and label from the catalog
// so the cashier knows which line to change.
type InsufficientStockError struct {
	VariantID    uuid.UUID
	ProductName  string
	VariantLabel string
}

// Description names the variant for the cashier: the product's name and,
// if the variant has one, its label ("Producto A (Rojo · M)").
func (e *InsufficientStockError) Description() string {
	if e.VariantLabel == "" {
		return e.ProductName
	}
	return e.ProductName + " (" + e.VariantLabel + ")"
}

// Error names the variant by id only: the message goes to logs and spans.
func (e *InsufficientStockError) Error() string {
	return ErrInsufficientStock.Error() + " of variant " + e.VariantID.String()
}

// Is makes the error match ErrInsufficientStock.
func (e *InsufficientStockError) Is(target error) bool { return target == ErrInsufficientStock }
