package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CashMovementKind says whether cash went into the drawer or out of it.
type CashMovementKind string

// Movement kinds: a cash in (more change, a payment received outside a
// sale) or a cash out (an expense, a withdrawal).
const (
	CashIn  CashMovementKind = "in"
	CashOut CashMovementKind = "out"
)

func (k CashMovementKind) String() string { return string(k) }

const maxCashMovementReasonLength = 200

// CashMovementInput is a movement being recorded by UserID.
type CashMovementInput struct {
	ID     uuid.UUID
	Kind   CashMovementKind
	Amount decimal.Decimal
	Reason string
	UserID uuid.UUID
	At     time.Time
}

// CashMovement is a cash in or out of a session, with its reason. It is
// immutable.
type CashMovement struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	sessionID uuid.UUID
	kind      CashMovementKind
	amount    decimal.Decimal
	reason    string
	userID    uuid.UUID
	at        time.Time
}

// newCashMovement checks a movement: a positive amount of COP (at most two
// decimals) and a reason (required, at most 200 characters, trimmed).
func newCashMovement(tenantID, sessionID uuid.UUID, in CashMovementInput) (CashMovement, error) {
	if in.ID == uuid.Nil {
		return CashMovement{}, ErrInvalidCashMovementID
	}
	if in.Kind != CashIn && in.Kind != CashOut {
		return CashMovement{}, ErrInvalidCashMovementKind
	}
	if !in.Amount.IsPositive() || !isMoney(in.Amount) {
		return CashMovement{}, ErrInvalidCashMovementAmount
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" || utf8.RuneCountInString(reason) > maxCashMovementReasonLength {
		return CashMovement{}, ErrInvalidCashMovementReason
	}
	if in.UserID == uuid.Nil {
		return CashMovement{}, ErrInvalidUserID
	}
	return CashMovement{
		id: in.ID, tenantID: tenantID, sessionID: sessionID, kind: in.Kind,
		amount: in.Amount, reason: reason, userID: in.UserID, at: in.At.UTC(),
	}, nil
}

// CashMovementSnapshot holds a stored movement's fields.
type CashMovementSnapshot struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	SessionID uuid.UUID
	Kind      CashMovementKind
	Amount    decimal.Decimal
	Reason    string
	UserID    uuid.UUID
	At        time.Time
}

// RehydrateCashMovement rebuilds a stored movement through the same checks.
func RehydrateCashMovement(snap CashMovementSnapshot) (CashMovement, error) {
	if snap.TenantID == uuid.Nil {
		return CashMovement{}, ErrInvalidTenantID
	}
	if snap.SessionID == uuid.Nil {
		return CashMovement{}, ErrInvalidCashSessionID
	}
	return newCashMovement(snap.TenantID, snap.SessionID, CashMovementInput{
		ID: snap.ID, Kind: snap.Kind, Amount: snap.Amount, Reason: snap.Reason, UserID: snap.UserID, At: snap.At,
	})
}

// ID returns the movement ID (UUID v7).
func (m CashMovement) ID() uuid.UUID { return m.id }

// TenantID returns the tenant the movement belongs to.
func (m CashMovement) TenantID() uuid.UUID { return m.tenantID }

// SessionID returns the session whose drawer it moved.
func (m CashMovement) SessionID() uuid.UUID { return m.sessionID }

// Kind returns whether it was a cash in or out.
func (m CashMovement) Kind() CashMovementKind { return m.kind }

// Amount returns how much moved, always positive.
func (m CashMovement) Amount() decimal.Decimal { return m.amount }

// Reason returns why it moved.
func (m CashMovement) Reason() string { return m.reason }

// UserID returns who recorded it.
func (m CashMovement) UserID() uuid.UUID { return m.userID }

// At returns when it was recorded, in UTC.
func (m CashMovement) At() time.Time { return m.at }
