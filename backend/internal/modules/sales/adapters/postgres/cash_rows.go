package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
)

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
	if err := insertCountAndTotals(ctx, q, s, c); err != nil {
		return err
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

// insertCountAndTotals stores the count and the payment totals of the
// closing c of s.
func insertCountAndTotals(ctx context.Context, q *sqlcgen.Queries, s *domain.CashSession, c *domain.CashClosing) error {
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
	// A closed session has every closing column.
	missing := []bool{
		row.ClosedAt == nil, row.ClosedBy == nil, row.CashReceived == nil, row.ChangeGiven == nil,
		row.CashIn == nil, row.CashOut == nil, row.SalesCount == nil, row.Expected == nil,
		row.Counted == nil, row.Difference == nil,
	}
	if slices.Contains(missing, true) {
		return nil, domain.ErrInconsistentCashSession
	}
	count, err := readCount(ctx, q, row.ID)
	if err != nil {
		return nil, err
	}
	activity, err := readClosedActivity(ctx, q, row)
	if err != nil {
		return nil, err
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

// readCount reads the count of a closed session.
func readCount(ctx context.Context, q *sqlcgen.Queries, id uuid.UUID) (domain.CashCount, error) {
	rows, err := q.ListCashSessionCounts(ctx, id)
	if err != nil {
		return domain.CashCount{}, err
	}
	counts := make([]domain.DenominationCount, 0, len(rows))
	for _, c := range rows {
		counts = append(counts, domain.DenominationCount{Denomination: c.Denomination, Quantity: c.Quantity})
	}
	return domain.NewCashCount(counts)
}

// readClosedActivity reads the activity a closed session stored, with its
// payment totals.
func readClosedActivity(ctx context.Context, q *sqlcgen.Queries, row sqlcgen.CashSession) (domain.CashActivity, error) {
	payments, err := q.ListCashSessionPayments(ctx, row.ID)
	if err != nil {
		return domain.CashActivity{}, err
	}
	activity := domain.CashActivity{
		CashReceived: *row.CashReceived, ChangeGiven: *row.ChangeGiven, CashIn: *row.CashIn, CashOut: *row.CashOut,
		Sales: int(*row.SalesCount),
	}
	for _, p := range payments {
		activity.ByMethod = append(activity.ByMethod, domain.MethodTotal{Method: domain.PaymentMethod(p.Method), Amount: p.Amount})
	}
	return activity, nil
}
