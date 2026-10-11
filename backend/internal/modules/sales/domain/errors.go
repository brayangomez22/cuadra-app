package domain

import "errors"

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

	ErrDiscountExceedsLimit = errors.New("sales: discount exceeds the user's limit")
	ErrSaleWithoutLines     = errors.New("sales: sale without lines")
	ErrInsufficientPayment  = errors.New("sales: payments do not cover the total")
	ErrNonCashOverpayment   = errors.New("sales: non-cash payments exceed the total")
	ErrSaleNotDraft         = errors.New("sales: sale is not a draft")

	ErrProductNotFound   = errors.New("sales: product not found")
	ErrProductInactive   = errors.New("sales: product is inactive")
	ErrInsufficientStock = errors.New("sales: insufficient stock")
)
