package domain

import "errors"

// Domain errors of the inventory module. The HTTP adapter maps them to status codes.
var (
	ErrInvalidTenantID     = errors.New("inventory: invalid tenant id")
	ErrInvalidProductID    = errors.New("inventory: invalid product id")
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

	ErrInsufficientStock        = errors.New("inventory: insufficient stock")
	ErrAdjustmentReasonRequired = errors.New("inventory: adjustment requires a reason")
	ErrAdjustmentReasonTooLong  = errors.New("inventory: adjustment reason is too long")
	ErrSameLocationTransfer     = errors.New("inventory: transfer to the same location")
	ErrTransferMismatch         = errors.New("inventory: transfer between different products or tenants")
)
