package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// CashSessionRepository implements domain.CashSessionRepository.
type CashSessionRepository struct{ db *db.DB }

// NewCashSessionRepository returns a CashSessionRepository on d.
func NewCashSessionRepository(d *db.DB) *CashSessionRepository { return &CashSessionRepository{db: d} }

// Open stores a new open session. The partial unique index
// cash_sessions_one_open_key keeps a user from having two.
func (r *CashSessionRepository) Open(ctx context.Context, s *domain.CashSession) error {
	err := r.db.WithTenantTx(ctx, s.TenantID(), func(tx pgx.Tx) error {
		return sqlcgen.New(tx).InsertCashSession(ctx, sqlcgen.InsertCashSessionParams{
			ID: s.ID(), TenantID: s.TenantID(), LocationID: s.LocationID(), UserID: s.UserID(),
			OpeningFloat: s.OpeningFloat(), OpenedAt: s.OpenedAt(),
		})
	})
	switch {
	case violates(err, uniqueViolation, cashSessionsOneOpenKey):
		return domain.ErrCashSessionAlreadyOpen
	case violates(err, foreignKeyViolation, cashSessionsLocationFKey):
		return domain.ErrLocationNotFound
	case err != nil:
		return fmt.Errorf("sales: open cash session %s: %w", s.ID(), err)
	}
	return nil
}

// Get returns a session with its closing, if closed.
func (r *CashSessionRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.CashSession, error) {
	var session *domain.CashSession
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		row, err := q.GetCashSession(ctx, id)
		if err != nil {
			return err
		}
		session, err = readCashSession(ctx, q, row)
		return err
	})
	if err != nil {
		return nil, cashError("get", id, err)
	}
	return session, nil
}

// Current returns the user's open session.
func (r *CashSessionRepository) Current(ctx context.Context, tenantID, userID uuid.UUID) (*domain.CashSession, error) {
	var session *domain.CashSession
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		row, err := q.CurrentCashSession(ctx, userID)
		if err != nil {
			return err
		}
		session, err = readCashSession(ctx, q, row)
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, domain.ErrCashSessionNotFound
	case err != nil:
		return nil, fmt.Errorf("sales: current cash session of user %s: %w", userID, err)
	}
	return session, nil
}

// Movements returns a session's movements, oldest first. A session of
// another tenant has none.
func (r *CashSessionRepository) Movements(ctx context.Context, tenantID, id uuid.UUID) ([]domain.CashMovement, error) {
	var out []domain.CashMovement
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := sqlcgen.New(tx).ListCashMovements(ctx, id)
		if err != nil {
			return err
		}
		out = make([]domain.CashMovement, 0, len(rows))
		for _, row := range rows {
			m, err := domain.RehydrateCashMovement(domain.CashMovementSnapshot{
				ID: row.ID, TenantID: row.TenantID, SessionID: row.SessionID, Kind: domain.CashMovementKind(row.Kind),
				Amount: row.Amount, Reason: row.Reason, UserID: row.UserID, At: row.CreatedAt,
			})
			if err != nil {
				return fmt.Errorf("cash movement %s: %w", row.ID, err)
			}
			out = append(out, m)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("sales: list movements of cash session %s: %w", id, err)
	}
	return out, nil
}

// AddMovement locks the session (FOR UPDATE), lets record check it and
// stores the movement.
func (r *CashSessionRepository) AddMovement(ctx context.Context, tenantID, id uuid.UUID, record func(*domain.CashSession) (domain.CashMovement, error)) (domain.CashMovement, error) {
	var recordErr error
	var m domain.CashMovement
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		row, err := q.LockCashSession(ctx, id)
		if err != nil {
			return err
		}
		session, err := readCashSession(ctx, q, row)
		if err != nil {
			return err
		}
		if m, err = record(session); err != nil {
			recordErr = err
			return err
		}
		return q.InsertCashMovement(ctx, sqlcgen.InsertCashMovementParams{
			ID: m.ID(), TenantID: m.TenantID(), SessionID: m.SessionID(), Kind: m.Kind().String(),
			Amount: m.Amount(), Reason: m.Reason(), UserID: m.UserID(), CreatedAt: m.At(),
		})
	})
	if recordErr != nil {
		return domain.CashMovement{}, recordErr
	}
	if err != nil {
		return domain.CashMovement{}, cashError("add movement to", id, err)
	}
	return m, nil
}

// EnsureOpen share-locks the session within the transaction carried by ctx:
// the sale's. The lock is held until that transaction ends, even though
// this call runs in a savepoint.
func (r *CashSessionRepository) EnsureOpen(ctx context.Context, tenantID, id uuid.UUID) error {
	var status string
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		status, err = sqlcgen.New(tx).ShareLockCashSession(ctx, id)
		return err
	})
	if err != nil {
		return cashError("lock", id, err)
	}
	if domain.CashSessionStatus(status) != domain.CashSessionOpen {
		return domain.ErrCashSessionClosed
	}
	return nil
}

// Close locks the session (FOR UPDATE, so it waits for the sales being
// stored in it), reads its activity, lets closeFn check it and stores the
// count, the payment totals and the closing, in that order: the triggers
// accept the count and the totals only while the session is open.
func (r *CashSessionRepository) Close(ctx context.Context, tenantID, id uuid.UUID, closeFn func(*domain.CashSession, domain.CashActivity) error) (*domain.CashSession, error) {
	var closeErr error
	var session *domain.CashSession
	err := r.db.WithTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		row, err := q.LockCashSession(ctx, id)
		if err != nil {
			return err
		}
		if session, err = readCashSession(ctx, q, row); err != nil {
			return err
		}
		activity, err := readActivity(ctx, q, id)
		if err != nil {
			return err
		}
		if err := closeFn(session, activity); err != nil {
			closeErr = err
			return err
		}
		return insertClosing(ctx, q, session)
	})
	if closeErr != nil {
		return nil, closeErr
	}
	if err != nil {
		return nil, cashError("close", id, err)
	}
	return session, nil
}

func readActivity(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (domain.CashActivity, error) {
	sales, err := q.CashSessionSales(ctx, &id)
	if err != nil {
		return domain.CashActivity{}, err
	}
	payments, err := q.CashSessionPayments(ctx, &id)
	if err != nil {
		return domain.CashActivity{}, err
	}
	movements, err := q.CashSessionMovementTotals(ctx, id)
	if err != nil {
		return domain.CashActivity{}, err
	}
	a := domain.CashActivity{
		CashReceived: decimal.Zero, ChangeGiven: sales.ChangeGiven,
		CashIn: movements.CashIn, CashOut: movements.CashOut, Sales: int(sales.SalesCount),
	}
	for _, p := range payments {
		amount := p.Amount
		if domain.PaymentMethod(p.Method) == domain.PaymentCash {
			a.CashReceived = p.Amount
			amount = p.Amount.Sub(sales.ChangeGiven)
		}
		a.ByMethod = append(a.ByMethod, domain.MethodTotal{Method: domain.PaymentMethod(p.Method), Amount: amount})
	}
	return a, nil
}

func insertClosing(ctx context.Context, q *sqlcgen.Queries, s *domain.CashSession) error {
	c := s.Closing()
	if c == nil {
		return fmt.Errorf("cash session %s was not closed", s.ID())
	}
	for _, d := range c.Count.Counts() {
		if err := q.InsertCashSessionCount(ctx, sqlcgen.InsertCashSessionCountParams{
			TenantID: s.TenantID(), SessionID: s.ID(), Denomination: d.Denomination, Quantity: d.Quantity,
		}); err != nil {
			return err
		}
	}
	for _, p := range c.Activity.ByMethod {
		if err := q.InsertCashSessionPayment(ctx, sqlcgen.InsertCashSessionPaymentParams{
			TenantID: s.TenantID(), SessionID: s.ID(), Method: p.Method.String(), Amount: p.Amount,
		}); err != nil {
			return err
		}
	}
	a := c.Activity
	if a.Sales < 0 || a.Sales > math.MaxInt32 {
		return errors.New("sales count of the cash session out of range")
	}
	salesCount := int32(a.Sales)
	var notes *string
	if c.Notes != "" {
		notes = &c.Notes
	}
	affected, err := q.CloseCashSession(ctx, sqlcgen.CloseCashSessionParams{
		ID: s.ID(), ClosedAt: &c.ClosedAt, ClosedBy: &c.ClosedBy, Notes: notes,
		CashReceived: &a.CashReceived, ChangeGiven: &a.ChangeGiven, CashIn: &a.CashIn, CashOut: &a.CashOut,
		SalesCount: &salesCount, Expected: &c.Expected, Counted: &c.Counted, Difference: &c.Difference,
	})
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("cash session %s is no longer open", s.ID())
	}
	return nil
}

// readCashSession rebuilds a session row, with its count and payment totals
// if it is closed.
func readCashSession(ctx context.Context, q *sqlcgen.Queries, row sqlcgen.CashSession) (*domain.CashSession, error) {
	snap := domain.CashSessionSnapshot{
		ID: row.ID, TenantID: row.TenantID, LocationID: row.LocationID, UserID: row.UserID,
		OpeningFloat: row.OpeningFloat, OpenedAt: row.OpenedAt, Status: domain.CashSessionStatus(row.Status),
	}
	if snap.Status == domain.CashSessionClosed {
		closing, err := readClosing(ctx, q, row)
		if err != nil {
			return nil, fmt.Errorf("cash session %s: %w", row.ID, err)
		}
		snap.Closing = closing
	}
	s, err := domain.RehydrateCashSession(snap)
	if err != nil {
		return nil, fmt.Errorf("cash session %s: %w", row.ID, err)
	}
	return s, nil
}

func readClosing(ctx context.Context, q *sqlcgen.Queries, row sqlcgen.CashSession) (*domain.CashClosing, error) {
	if row.ClosedAt == nil || row.ClosedBy == nil || row.CashReceived == nil || row.ChangeGiven == nil ||
		row.CashIn == nil || row.CashOut == nil || row.SalesCount == nil || row.Expected == nil ||
		row.Counted == nil || row.Difference == nil {
		return nil, domain.ErrInconsistentCashSession
	}
	countRows, err := q.ListCashSessionCounts(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	counts := make([]domain.DenominationCount, 0, len(countRows))
	for _, c := range countRows {
		counts = append(counts, domain.DenominationCount{Denomination: c.Denomination, Quantity: c.Quantity})
	}
	count, err := domain.NewCashCount(counts)
	if err != nil {
		return nil, err
	}
	paymentRows, err := q.ListCashSessionPayments(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	activity := domain.CashActivity{
		CashReceived: *row.CashReceived, ChangeGiven: *row.ChangeGiven, CashIn: *row.CashIn, CashOut: *row.CashOut,
		Sales: int(*row.SalesCount),
	}
	for _, p := range paymentRows {
		activity.ByMethod = append(activity.ByMethod, domain.MethodTotal{Method: domain.PaymentMethod(p.Method), Amount: p.Amount})
	}
	c := &domain.CashClosing{
		ClosedBy: *row.ClosedBy, ClosedAt: *row.ClosedAt, Count: count, Activity: activity,
		Expected: *row.Expected, Counted: *row.Counted, Difference: *row.Difference,
	}
	if row.Notes != nil {
		c.Notes = *row.Notes
	}
	return c, nil
}

// cashError translates a missing row (another tenant's session included) to
// ErrCashSessionNotFound and wraps the rest.
func cashError(op string, id uuid.UUID, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrCashSessionNotFound
	}
	return fmt.Errorf("sales: %s cash session %s: %w", op, id, err)
}
