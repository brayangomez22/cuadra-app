package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SaleRepository persists sales. Every operation is scoped to a tenant: a
// sale of another tenant is reported as ErrSaleNotFound.
type SaleRepository interface {
	// SaveConfirmed stores a confirmed sale and calls reserve within the
	// same transaction, so the sale and its stock exits are stored together
	// or not at all. If reserve fails, nothing is stored and its error is
	// returned. It assigns the sale the next number of its location, without
	// gaps. ErrLocationNotFound if the location does not exist in the
	// tenant.
	SaveConfirmed(ctx context.Context, s *Sale, reserve func(context.Context) error) error
	// Get returns a sale with its lines and payments.
	Get(ctx context.Context, tenantID, id uuid.UUID) (*Sale, error)
	// List returns one page of the sales confirmed in [From, To), newest
	// first.
	List(ctx context.Context, tenantID uuid.UUID, f SaleFilter) (SalePage, error)
	// Void locks a sale, calls change with it (which voids it and gives its
	// stock back, within the same transaction through ctx) and stores the
	// voided sale. If change fails, nothing is stored and its error is
	// returned. Two concurrent calls for a sale run one after the other.
	Void(ctx context.Context, tenantID, id uuid.UUID, change func(context.Context, *Sale) error) (*Sale, error)
}

// SaleFilter selects the sales confirmed in [From, To), optionally at one
// location.
type SaleFilter struct {
	From       time.Time
	To         time.Time
	LocationID *uuid.UUID
	Limit      int
	Offset     int
}

// SaleSummary is a sale as the day's list shows it, without lines.
type SaleSummary struct {
	ID          uuid.UUID
	Number      int64
	LocationID  uuid.UUID
	UserID      uuid.UUID
	Status      SaleStatus
	Total       decimal.Decimal
	ConfirmedAt time.Time
	VoidedAt    *time.Time
}

// SummaryOf returns the summary of a stored sale.
func SummaryOf(s *Sale) SaleSummary {
	sum := SaleSummary{
		ID: s.id, Number: s.number, LocationID: s.locationID, UserID: s.userID,
		Status: s.status, Total: s.Totals().Total,
	}
	if s.confirmedAt != nil {
		sum.ConfirmedAt = *s.confirmedAt
	}
	if s.void != nil {
		at := s.void.At
		sum.VoidedAt = &at
	}
	return sum
}

// SalePage is one page of sales and how many match in total.
type SalePage struct {
	Items []SaleSummary
	Total int
}

// PolicyRepository reads the tenant's sales rules.
type PolicyRepository interface {
	// Get returns the tenant's policy, or the zero SalesPolicy if it has not
	// set one.
	Get(ctx context.Context, tenantID uuid.UUID) (SalesPolicy, error)
}

// CashSessionRepository persists cash sessions and their movements. Every
// operation is scoped to a tenant: a session of another tenant is reported as
// ErrCashSessionNotFound.
type CashSessionRepository interface {
	// Open stores a new open session. ErrCashSessionAlreadyOpen if its user
	// already has one open; ErrLocationNotFound if the location does not
	// exist in the tenant.
	Open(ctx context.Context, s *CashSession) error
	// Get returns a session, open or closed.
	Get(ctx context.Context, tenantID, id uuid.UUID) (*CashSession, error)
	// Current returns the user's open session, or ErrCashSessionNotFound.
	Current(ctx context.Context, tenantID, userID uuid.UUID) (*CashSession, error)
	// Movements returns a session's movements, oldest first.
	Movements(ctx context.Context, tenantID, id uuid.UUID) ([]CashMovement, error)
	// AddMovement locks a session, calls record with it and stores the
	// movement record returns. If record fails, nothing is stored and its
	// error is returned.
	AddMovement(ctx context.Context, tenantID, id uuid.UUID, record func(*CashSession) (CashMovement, error)) (CashMovement, error)
	// EnsureOpen locks a session for share until the transaction carried by
	// ctx ends (the one storing a sale) and fails with ErrCashSessionClosed
	// if it is closed. A closing waits for the sales that hold the lock.
	EnsureOpen(ctx context.Context, tenantID, id uuid.UUID) error
	// Close locks a session, reads its activity (the confirmed sales charged
	// in it and its movements) and calls closeFn with both, then stores the
	// closing. If closeFn fails, nothing is stored and its error is returned.
	Close(ctx context.Context, tenantID, id uuid.UUID, closeFn func(*CashSession, CashActivity) error) (*CashSession, error)
}
