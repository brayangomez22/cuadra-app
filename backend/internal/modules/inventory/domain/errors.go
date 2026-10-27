package domain

import "errors"

// Domain errors of the inventory module. The HTTP adapter maps them to status codes.
var (
	ErrInvalidTenantID     = errors.New("inventory: invalid tenant id")
	ErrInvalidVariantID    = errors.New("inventory: invalid variant id")
	ErrInvalidLocationID   = errors.New("inventory: invalid location id")
	ErrInvalidMovementID   = errors.New("inventory: invalid movement id")
	ErrInvalidUserID       = errors.New("inventory: invalid user id")
	ErrInvalidLocationName = errors.New("inventory: invalid location name")
	ErrInvalidMovementType = errors.New("inventory: invalid movement type")
	ErrInvalidQuantity     = errors.New("inventory: invalid quantity")
	ErrInvalidUnitCost     = errors.New("inventory: invalid unit cost")
	ErrInvalidReference    = errors.New("inventory: invalid document reference")
	ErrInvalidStockLevel   = errors.New("inventory: invalid stock level")
	ErrInvalidMovement     = errors.New("inventory: invalid stock movement")
	ErrInvalidReorderRange = errors.New("inventory: reorder minimum and maximum must satisfy 0 <= min <= max")

	ErrInsufficientStock        = errors.New("inventory: insufficient stock")
	ErrAdjustmentReasonRequired = errors.New("inventory: adjustment requires a reason")
	ErrAdjustmentReasonTooLong  = errors.New("inventory: adjustment reason is too long")
	ErrSameLocationTransfer     = errors.New("inventory: transfer to the same location")
	ErrTransferMismatch         = errors.New("inventory: transfer between different variants or tenants")
	ErrLocationInactive         = errors.New("inventory: location is inactive")
	ErrSaleAlreadyReversed      = errors.New("inventory: the sale's stock was already given back")

	ErrLocationNotFound    = errors.New("inventory: location not found")
	ErrLocationNameTaken   = errors.New("inventory: location name taken")
	ErrVariantNotFound     = errors.New("inventory: variant not found")
	ErrInvalidUnitFactor   = errors.New("inventory: invalid unit factor")
	ErrQuantityPrecision   = errors.New("inventory: quantity in the base unit needs more than four decimals")
	ErrUnitNotAvailable    = errors.New("inventory: the variant has no such alternate unit")
	ErrReorderRuleNotFound = errors.New("inventory: reorder rule not found")
)
