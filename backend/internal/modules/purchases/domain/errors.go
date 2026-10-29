package domain

import (
	"errors"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/taxid"
)

// Domain errors of the purchases module. The HTTP adapter maps them to
// status codes.
var (
	ErrInvalidTenantID   = errors.New("purchases: invalid tenant id")
	ErrInvalidSupplierID = errors.New("purchases: invalid supplier id")
	ErrInvalidPurchaseID = errors.New("purchases: invalid purchase id")
	// ErrInvalidNIT and ErrInvalidNITCheckDigit are the shared kernel's.
	ErrInvalidNIT           = taxid.ErrInvalidNIT
	ErrInvalidNITCheckDigit = taxid.ErrInvalidNITCheckDigit
	ErrInvalidSupplierName  = errors.New("purchases: invalid supplier name")
	ErrInvalidContact       = errors.New("purchases: invalid contact data")
	ErrInvalidPaymentTerm   = errors.New("purchases: payment term must be between 0 and 365 days")

	ErrInvalidLocationID    = errors.New("purchases: invalid location id")
	ErrInvalidUserID        = errors.New("purchases: invalid user id")
	ErrInvalidInvoiceNumber = errors.New("purchases: invalid invoice number")
	ErrInvalidNotes         = errors.New("purchases: notes are too long")
	ErrInvalidPayment       = errors.New("purchases: invalid payment")
	ErrInvoiceDateInFuture  = errors.New("purchases: the invoice date is after today")
	ErrDueBeforeIssue       = errors.New("purchases: the due date is before the invoice date")
	ErrNoLines              = errors.New("purchases: a purchase needs at least one line")
	ErrTooManyLines         = errors.New("purchases: too many lines")
	ErrInvalidVariantID     = errors.New("purchases: invalid variant id")
	ErrInvalidProductName   = errors.New("purchases: invalid product name")
	ErrInvalidUnit          = errors.New("purchases: invalid purchase unit")
	ErrInvalidQuantity      = errors.New("purchases: invalid quantity")
	ErrInvalidUnitCost      = errors.New("purchases: invalid unit cost")
	ErrInvalidTaxRate       = errors.New("purchases: invalid tax rate")
	ErrReceiptMismatch      = errors.New("purchases: the receipt does not match the purchase's lines")
	ErrInvalidPurchase      = errors.New("purchases: invalid stored purchase")

	// Errors reported by the repositories, the use cases and the ports to
	// other modules.
	ErrSupplierNotFound  = errors.New("purchases: supplier not found")
	ErrSupplierInactive  = errors.New("purchases: supplier is inactive")
	ErrNITTaken          = errors.New("purchases: NIT already used by another supplier of the tenant")
	ErrInvoiceTaken      = errors.New("purchases: the supplier's invoice is already registered")
	ErrPurchaseNotFound  = errors.New("purchases: purchase not found")
	ErrVariantNotFound   = errors.New("purchases: variant not found")
	ErrVariantInactive   = errors.New("purchases: variant is inactive")
	ErrUnitNotAvailable  = errors.New("purchases: the variant has no such unit")
	ErrQuantityPrecision = errors.New("purchases: quantity in the base unit needs more than four decimals")
	ErrLocationNotFound  = errors.New("purchases: location not found")
	ErrLocationInactive  = errors.New("purchases: location is inactive")
)
