package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const maxReasonLength = 500

// MovementType is the kind of change a StockMovement records.
type MovementType string

// Movement types. Inbound types have a positive quantity, outbound types a
// negative one; an adjustment can be either.
const (
	MovementPurchaseIn  MovementType = "purchase_in"
	MovementSaleOut     MovementType = "sale_out"
	MovementAdjustment  MovementType = "adjustment"
	MovementTransferOut MovementType = "transfer_out"
	MovementTransferIn  MovementType = "transfer_in"
)

// ParseMovementType returns the movement type with code s. Matching is exact.
func ParseMovementType(s string) (MovementType, error) {
	t := MovementType(s)
	switch t {
	case MovementPurchaseIn, MovementSaleOut, MovementAdjustment, MovementTransferOut, MovementTransferIn:
		return t, nil
	}
	return "", ErrInvalidMovementType
}

func (t MovementType) String() string { return string(t) }

func (t MovementType) isTransfer() bool {
	return t == MovementTransferOut || t == MovementTransferIn
}

// validQuantity reports whether q has the sign the movement type requires.
func (t MovementType) validQuantity(q decimal.Decimal) bool {
	switch t {
	case MovementPurchaseIn, MovementTransferIn:
		return q.IsPositive()
	case MovementSaleOut, MovementTransferOut:
		return q.IsNegative()
	default:
		return !q.IsZero()
	}
}

// DocumentKind is the kind of document that caused a movement.
type DocumentKind string

// Document kinds. DocumentTransfer is reserved for the pair of movements
// that Transfer creates.
const (
	DocumentPurchase DocumentKind = "purchase"
	DocumentSale     DocumentKind = "sale"
	DocumentTransfer DocumentKind = "transfer"
)

// DocumentRef points to the document that caused a movement (a sale, a
// purchase, a transfer). It is a reference only: inventory does not know
// those documents.
type DocumentRef struct {
	Kind DocumentKind
	ID   uuid.UUID
}

func (r DocumentRef) valid() bool {
	switch r.Kind {
	case DocumentPurchase, DocumentSale, DocumentTransfer:
		return r.ID != uuid.Nil
	}
	return false
}

func copyRef(r *DocumentRef) *DocumentRef {
	if r == nil {
		return nil
	}
	c := *r
	return &c
}

// MovementInfo is who registers a movement, why and when. Reference is
// optional.
type MovementInfo struct {
	UserID     uuid.UUID
	Reference  *DocumentRef
	OccurredAt time.Time
}

// StockMovement is one immutable entry of the kardex: a change of a product's
// stock at a location. It is created only by StockLevel (and Transfer), so it
// always agrees with the level it changed. Quantity is signed: positive in,
// negative out. BalanceAfter and AverageCostAfter are the level's state right
// after the movement, so the kardex shows running totals without replaying
// history.
type StockMovement struct {
	id               uuid.UUID
	tenantID         uuid.UUID
	productID        uuid.UUID
	locationID       uuid.UUID
	movementType     MovementType
	quantity         decimal.Decimal
	unitCost         decimal.Decimal
	balanceAfter     decimal.Decimal
	averageCostAfter decimal.Decimal
	reason           string
	reference        *DocumentRef
	userID           uuid.UUID
	occurredAt       time.Time
}

// newMovement checks the invariants shared by the movements StockLevel
// creates and RehydrateStockMovement.
func newMovement(s StockMovementSnapshot) (*StockMovement, error) {
	if s.ID == uuid.Nil {
		return nil, ErrInvalidMovementID
	}
	if s.TenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	if s.ProductID == uuid.Nil {
		return nil, ErrInvalidProductID
	}
	if s.LocationID == uuid.Nil {
		return nil, ErrInvalidLocationID
	}
	if s.UserID == uuid.Nil {
		return nil, ErrInvalidUserID
	}
	if _, err := ParseMovementType(string(s.Type)); err != nil {
		return nil, err
	}
	if !fitsStorage(s.Quantity) || !s.Type.validQuantity(s.Quantity) ||
		!validCost(s.UnitCost) || !fitsStorage(s.BalanceAfter) || !validCost(s.AverageCostAfter) {
		return nil, ErrInvalidMovement
	}
	reason := strings.TrimSpace(s.Reason)
	if s.Type == MovementAdjustment {
		var err error
		if reason, err = normalizeReason(reason); err != nil {
			return nil, err
		}
	} else if reason != "" {
		return nil, ErrInvalidMovement
	}
	if s.Reference != nil && !s.Reference.valid() {
		return nil, ErrInvalidReference
	}
	isTransferRef := s.Reference != nil && s.Reference.Kind == DocumentTransfer
	if isTransferRef != s.Type.isTransfer() {
		return nil, ErrInvalidReference
	}
	return &StockMovement{
		id:               s.ID,
		tenantID:         s.TenantID,
		productID:        s.ProductID,
		locationID:       s.LocationID,
		movementType:     s.Type,
		quantity:         s.Quantity,
		unitCost:         s.UnitCost,
		balanceAfter:     s.BalanceAfter,
		averageCostAfter: s.AverageCostAfter,
		reason:           reason,
		reference:        copyRef(s.Reference),
		userID:           s.UserID,
		occurredAt:       s.OccurredAt.UTC(),
	}, nil
}

// normalizeReason trims an adjustment reason; it is required and at most
// maxReasonLength runes.
func normalizeReason(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ErrAdjustmentReasonRequired
	}
	if utf8.RuneCountInString(s) > maxReasonLength {
		return "", ErrAdjustmentReasonTooLong
	}
	return s, nil
}

// ID returns the movement ID (UUID v7).
func (m *StockMovement) ID() uuid.UUID { return m.id }

// TenantID returns the tenant the movement belongs to.
func (m *StockMovement) TenantID() uuid.UUID { return m.tenantID }

// ProductID returns the product whose stock changed.
func (m *StockMovement) ProductID() uuid.UUID { return m.productID }

// LocationID returns the location whose stock changed.
func (m *StockMovement) LocationID() uuid.UUID { return m.locationID }

// Type returns the movement type.
func (m *StockMovement) Type() MovementType { return m.movementType }

// Quantity returns the signed quantity: positive in, negative out.
func (m *StockMovement) Quantity() decimal.Decimal { return m.quantity }

// UnitCost returns the cost per unit the movement was valued at: the
// purchase cost for entries, the average cost otherwise.
func (m *StockMovement) UnitCost() decimal.Decimal { return m.unitCost }

// BalanceAfter returns the stock at the location right after the movement.
func (m *StockMovement) BalanceAfter() decimal.Decimal { return m.balanceAfter }

// AverageCostAfter returns the weighted average cost right after the movement.
func (m *StockMovement) AverageCostAfter() decimal.Decimal { return m.averageCostAfter }

// Reason returns why an adjustment was made; empty for other types.
func (m *StockMovement) Reason() string { return m.reason }

// Reference returns a copy of the document reference, or nil if none.
func (m *StockMovement) Reference() *DocumentRef { return copyRef(m.reference) }

// UserID returns who registered the movement.
func (m *StockMovement) UserID() uuid.UUID { return m.userID }

// OccurredAt returns when the movement happened, in UTC.
func (m *StockMovement) OccurredAt() time.Time { return m.occurredAt }

// StockMovementSnapshot holds a stored movement's fields, for RehydrateStockMovement.
type StockMovementSnapshot struct {
	ID               uuid.UUID
	TenantID         uuid.UUID
	ProductID        uuid.UUID
	LocationID       uuid.UUID
	Type             MovementType
	Quantity         decimal.Decimal
	UnitCost         decimal.Decimal
	BalanceAfter     decimal.Decimal
	AverageCostAfter decimal.Decimal
	Reason           string
	Reference        *DocumentRef
	UserID           uuid.UUID
	OccurredAt       time.Time
}

// RehydrateStockMovement rebuilds a stored movement, checking the same
// invariants as the movements StockLevel creates.
func RehydrateStockMovement(s StockMovementSnapshot) (*StockMovement, error) { return newMovement(s) }
