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
	ErrInsufficientStock = errors.New("sales: insufficient stock")
)

// InsufficientStockError is ErrInsufficientStock for one product of the
// sale: errors.Is(err, ErrInsufficientStock) holds. The StockReserver fills
// ProductID; ConfirmSale adds the name from the catalog so the cashier knows
// which product to take off.
type InsufficientStockError struct {
	ProductID   uuid.UUID
	ProductName string
}

// Error names the product by id only: the message goes to logs and spans.
func (e *InsufficientStockError) Error() string {
	return ErrInsufficientStock.Error() + " of product " + e.ProductID.String()
}

// Is makes the error match ErrInsufficientStock.
func (e *InsufficientStockError) Is(target error) bool { return target == ErrInsufficientStock }
