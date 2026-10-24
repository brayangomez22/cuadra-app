package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

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
	case db.Violates(err, db.UniqueViolation, cashSessionsOneOpenKey):
		return domain.ErrCashSessionAlreadyOpen
	case db.Violates(err, db.ForeignKeyViolation, cashSessionsLocationFKey):
		return domain.ErrLocationNotFound
	case err != nil:
		return fmt.Errorf("sales: open cash session %s: %w", s.ID(), err)
	}
	return nil
}

// Get returns a session with its closing, if closed.
func (r *CashSessionRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.CashSession, error) {
	session, err := r.readOne(ctx, tenantID, func(q *sqlcgen.Queries) (sqlcgen.CashSession, error) { return q.GetCashSession(ctx, id) })
	if err != nil {
		return nil, cashError("get", id, err)
	}
	return session, nil
}

// readOne reads the session row get returns, with its closing.
func (r *CashSessionRepository) readOne(
	ctx context.Context, tenantID uuid.UUID, get func(*sqlcgen.Queries) (sqlcgen.CashSession, error),
) (*domain.CashSession, error) {
	return db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (*domain.CashSession, error) {
		q := sqlcgen.New(tx)
		row, err := get(q)
		if err != nil {
			return nil, err
		}
		return readCashSession(ctx, q, row)
	})
}

// Current returns the user's open session.
func (r *CashSessionRepository) Current(ctx context.Context, tenantID, userID uuid.UUID) (*domain.CashSession, error) {
	session, err := r.readOne(ctx, tenantID, func(q *sqlcgen.Queries) (sqlcgen.CashSession, error) {
		return q.CurrentCashSession(ctx, userID)
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
	status, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (string, error) {
		return sqlcgen.New(tx).ShareLockCashSession(ctx, id)
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

// cashError translates a missing row (another tenant's session included) to
// ErrCashSessionNotFound and wraps the rest.
func cashError(op string, id uuid.UUID, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrCashSessionNotFound
	}
	return fmt.Errorf("sales: %s cash session %s: %w", op, id, err)
}
