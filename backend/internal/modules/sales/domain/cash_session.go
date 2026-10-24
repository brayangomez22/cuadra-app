package domain

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CashSessionStatus is the state of a cash session.
type CashSessionStatus string

// Cash session states. A session is opened with its float and closed once,
// with the count of the drawer; a closed session never changes.
const (
	CashSessionOpen   CashSessionStatus = "open"
	CashSessionClosed CashSessionStatus = "closed"
)

func (s CashSessionStatus) String() string { return string(s) }

// CashResult is how the count of a closing compares with the expected cash.
type CashResult string

// Closing results: the count matches, falls short or is over.
const (
	CashExact    CashResult = "exact"
	CashShortage CashResult = "shortage"
	CashOverage  CashResult = "overage"
)

func (r CashResult) String() string { return string(r) }

const maxClosingNotesLength = 500

// NewCashSessionParams are the fields of a session being opened.
type NewCashSessionParams struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	LocationID uuid.UUID
	// UserID is the cashier: only they sell, record movements and close it.
	UserID uuid.UUID
	// OpeningFloat is the cash in the drawer when it opens (la base).
	OpeningFloat decimal.Decimal
	OpenedAt     time.Time
}

// MethodTotal is what the sales of a session collected with one payment
// method. For cash it is net of the change given.
type MethodTotal struct {
	Method PaymentMethod
	Amount decimal.Decimal
}

// CashActivity is what went through a session's drawer: the cash tendered in
// its confirmed sales and the change given back, the cash ins and outs, and,
// for the report, how many sales and what each payment method collected.
type CashActivity struct {
	CashReceived decimal.Decimal
	ChangeGiven  decimal.Decimal
	CashIn       decimal.Decimal
	CashOut      decimal.Decimal
	Sales        int
	ByMethod     []MethodTotal
}

func (a CashActivity) valid() bool {
	for _, d := range []decimal.Decimal{a.CashReceived, a.ChangeGiven, a.CashIn, a.CashOut} {
		if d.IsNegative() || !isMoney(d) {
			return false
		}
	}
	return a.Sales >= 0
}

// ExpectedCash is the cash the drawer should hold: the opening float plus
// the cash received in sales and the cash ins, minus the cash outs and the
// change given. It is negative if more went out than there was.
func ExpectedCash(openingFloat decimal.Decimal, a CashActivity) decimal.Decimal {
	return openingFloat.Add(a.CashReceived).Add(a.CashIn).Sub(a.CashOut).Sub(a.ChangeGiven)
}

// CloseCashInput is a closing: who closes, the count of the drawer, the
// session's activity (which the repository reads while the session is
// locked) and optional notes.
type CloseCashInput struct {
	UserID   uuid.UUID
	Count    CashCount
	Activity CashActivity
	Notes    string
	At       time.Time
}

// CashClosing is the closing of a session. Expected is ExpectedCash of the
// activity, Counted the count's total and Difference = Counted − Expected:
// negative is a shortage, positive an overage.
type CashClosing struct {
	ClosedBy   uuid.UUID
	ClosedAt   time.Time
	Notes      string
	Count      CashCount
	Activity   CashActivity
	Expected   decimal.Decimal
	Counted    decimal.Decimal
	Difference decimal.Decimal
}

// Result says whether the count matched the expected cash, fell short or was
// over.
func (c CashClosing) Result() CashResult {
	switch c.Difference.Sign() {
	case -1:
		return CashShortage
	case 1:
		return CashOverage
	}
	return CashExact
}

// CashSession is a cashier's drawer at a location, from its opening to its
// closing with the count (arqueo). The sales the cashier confirms while it is
// open are charged in it. A closed session is immutable.
type CashSession struct {
	id           uuid.UUID
	tenantID     uuid.UUID
	locationID   uuid.UUID
	userID       uuid.UUID
	openingFloat decimal.Decimal
	openedAt     time.Time
	status       CashSessionStatus
	closing      *CashClosing
}

// OpenCashSession opens a session with an opening float of zero or more COP
// (at most two decimals).
func OpenCashSession(p NewCashSessionParams) (*CashSession, error) {
	switch {
	case p.ID == uuid.Nil:
		return nil, ErrInvalidCashSessionID
	case p.TenantID == uuid.Nil:
		return nil, ErrInvalidTenantID
	case p.LocationID == uuid.Nil:
		return nil, ErrInvalidLocationID
	case p.UserID == uuid.Nil:
		return nil, ErrInvalidUserID
	case p.OpeningFloat.IsNegative() || !isMoney(p.OpeningFloat):
		return nil, ErrInvalidOpeningFloat
	}
	return &CashSession{
		id: p.ID, tenantID: p.TenantID, locationID: p.LocationID, userID: p.UserID,
		openingFloat: p.OpeningFloat, openedAt: p.OpenedAt.UTC(), status: CashSessionOpen,
	}, nil
}

// RecordMovement records a cash in or out of an open session. Only the
// session's cashier records them.
func (s *CashSession) RecordMovement(in CashMovementInput) (CashMovement, error) {
	if s.status != CashSessionOpen {
		return CashMovement{}, ErrCashSessionClosed
	}
	if in.UserID != s.userID {
		return CashMovement{}, ErrCashSessionNotOwner
	}
	return newCashMovement(s.tenantID, s.id, in)
}

// Close closes an open session with the count of the drawer and the
// session's activity, computing the expected cash and the difference. Only
// the session's cashier closes it, once. On error the session is unchanged.
func (s *CashSession) Close(in CloseCashInput) error {
	if s.status != CashSessionOpen {
		return ErrCashSessionClosed
	}
	if in.UserID != s.userID {
		return ErrCashSessionNotOwner
	}
	notes := strings.TrimSpace(in.Notes)
	if utf8.RuneCountInString(notes) > maxClosingNotesLength {
		return ErrInvalidCashNotes
	}
	if !in.Activity.valid() {
		return ErrInvalidCashActivity
	}
	expected := ExpectedCash(s.openingFloat, in.Activity)
	counted := in.Count.Total()
	activity := in.Activity
	activity.ByMethod = slices.Clone(in.Activity.ByMethod)
	s.closing = &CashClosing{
		ClosedBy: in.UserID, ClosedAt: in.At.UTC(), Notes: notes, Count: in.Count, Activity: activity,
		Expected: expected, Counted: counted, Difference: counted.Sub(expected),
	}
	s.status = CashSessionClosed
	return nil
}

// ID returns the session ID (UUID v7).
func (s *CashSession) ID() uuid.UUID { return s.id }

// TenantID returns the tenant the session belongs to.
func (s *CashSession) TenantID() uuid.UUID { return s.tenantID }

// LocationID returns where the drawer is: the only location its sales can
// happen at.
func (s *CashSession) LocationID() uuid.UUID { return s.locationID }

// UserID returns the cashier.
func (s *CashSession) UserID() uuid.UUID { return s.userID }

// OpeningFloat returns the cash the drawer started with.
func (s *CashSession) OpeningFloat() decimal.Decimal { return s.openingFloat }

// OpenedAt returns when the session was opened, in UTC.
func (s *CashSession) OpenedAt() time.Time { return s.openedAt }

// Status returns the session's state.
func (s *CashSession) Status() CashSessionStatus { return s.status }

// Closing returns a copy of the closing, or nil while the session is open.
func (s *CashSession) Closing() *CashClosing {
	if s.closing == nil {
		return nil
	}
	c := *s.closing
	c.Activity.ByMethod = slices.Clone(c.Activity.ByMethod)
	return &c
}

// CashSessionSnapshot holds a stored session's fields, for
// RehydrateCashSession. Closing is set only for a closed session.
type CashSessionSnapshot struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	LocationID   uuid.UUID
	UserID       uuid.UUID
	OpeningFloat decimal.Decimal
	OpenedAt     time.Time
	Status       CashSessionStatus
	Closing      *CashClosing
}

// Snapshot returns the session's fields as the repository stores them.
func (s *CashSession) Snapshot() CashSessionSnapshot {
	return CashSessionSnapshot{
		ID: s.id, TenantID: s.tenantID, LocationID: s.locationID, UserID: s.userID,
		OpeningFloat: s.openingFloat, OpenedAt: s.openedAt, Status: s.status, Closing: s.Closing(),
	}
}

// RehydrateCashSession rebuilds a stored session through the same rules
// that built it. A closed session whose stored expected, counted or
// difference are not the ones its float, activity and count give fails with
// ErrInconsistentCashSession.
func RehydrateCashSession(snap CashSessionSnapshot) (*CashSession, error) {
	if snap.Status != CashSessionOpen && snap.Status != CashSessionClosed {
		return nil, ErrInconsistentCashSession
	}
	if (snap.Status == CashSessionClosed) != (snap.Closing != nil) {
		return nil, ErrInconsistentCashSession
	}
	s, err := OpenCashSession(NewCashSessionParams{
		ID: snap.ID, TenantID: snap.TenantID, LocationID: snap.LocationID, UserID: snap.UserID,
		OpeningFloat: snap.OpeningFloat, OpenedAt: snap.OpenedAt,
	})
	if err != nil {
		return nil, err
	}
	if snap.Closing == nil {
		return s, nil
	}
	if err := s.rehydrateClosing(*snap.Closing); err != nil {
		return nil, err
	}
	return s, nil
}

// rehydrateClosing closes the session with the stored count and checks the
// stored expected, counted and difference.
func (s *CashSession) rehydrateClosing(c CashClosing) error {
	count, err := NewCashCount(c.Count.Counts())
	if err != nil {
		return err
	}
	if err := s.Close(CloseCashInput{UserID: c.ClosedBy, Count: count, Activity: c.Activity, Notes: c.Notes, At: c.ClosedAt}); err != nil {
		return err
	}
	got := s.closing
	if !got.Expected.Equal(c.Expected) || !got.Counted.Equal(c.Counted) || !got.Difference.Equal(c.Difference) {
		return ErrInconsistentCashSession
	}
	return nil
}
