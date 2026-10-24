package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// OpenCashSessionInput opens the actor's cash session at a location with an
// opening float.
type OpenCashSessionInput struct {
	LocationID   uuid.UUID
	OpeningFloat decimal.Decimal
}

// CashMovementRequest is a cash in or out; Kind is a domain.CashMovementKind
// code.
type CashMovementRequest struct {
	Kind   string
	Amount decimal.Decimal
	Reason string
}

// CloseCashSessionInput is the count of the drawer by denomination and
// optional notes.
type CloseCashSessionInput struct {
	Counts []domain.DenominationCount
	Notes  string
}

// CashSessionView is a session with its movements, oldest first.
type CashSessionView struct {
	Session   *domain.CashSession
	Movements []domain.CashMovement
}

// CurrentCashSession is whether the tenant requires an open session to sell
// and the actor's open session, if any.
type CurrentCashSession struct {
	Required bool
	Session  *CashSessionView
}

// OpenCashSession opens the actor's cash session. A user has at most one
// open session.
func (s *Service) OpenCashSession(ctx context.Context, a Actor, in OpenCashSessionInput) (_ *domain.CashSession, err error) {
	ctx, span := s.startSpan(ctx, "OpenCashSession", a, attribute.String("location.id", in.LocationID.String()))
	defer func() { usecase.End(span, err) }()

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	span.SetAttributes(cashSessionAttr(id))
	session, err := domain.OpenCashSession(domain.NewCashSessionParams{
		ID: id, TenantID: a.TenantID, LocationID: in.LocationID, UserID: a.UserID,
		OpeningFloat: in.OpeningFloat, OpenedAt: s.Now(),
	})
	if err != nil {
		return nil, err
	}
	if err := s.CashSessions.Open(ctx, session); err != nil {
		return nil, err
	}
	s.cashOpened.Add(ctx, 1)
	s.Logger.InfoContext(ctx, "cash session opened",
		slog.String("cash_session_id", id.String()), slog.String("location_id", in.LocationID.String()))
	return session, nil
}

// RecordCashMovement records a cash in or out of the actor's open session.
func (s *Service) RecordCashMovement(ctx context.Context, a Actor, sessionID uuid.UUID, in CashMovementRequest) (_ domain.CashMovement, err error) {
	ctx, span := s.startSpan(ctx, "RecordCashMovement", a, cashSessionAttr(sessionID))
	defer func() { usecase.End(span, err) }()

	id, err := uuid.NewV7()
	if err != nil {
		return domain.CashMovement{}, err
	}
	m, err := s.CashSessions.AddMovement(ctx, a.TenantID, sessionID, func(session *domain.CashSession) (domain.CashMovement, error) {
		return session.RecordMovement(domain.CashMovementInput{
			ID: id, Kind: domain.CashMovementKind(in.Kind), Amount: in.Amount, Reason: in.Reason,
			UserID: a.UserID, At: s.Now(),
		})
	})
	if err != nil {
		return domain.CashMovement{}, err
	}
	s.cashMovements.Add(ctx, 1, metric.WithAttributes(attribute.String("cash.movement_kind", m.Kind().String())))
	return m, nil
}

// CloseCashSession closes the actor's open session with the count of the
// drawer. The expected cash comes from the confirmed sales charged in it and
// its movements, read while the session is locked, so no sale slips in
// between. The closing never changes afterwards.
func (s *Service) CloseCashSession(ctx context.Context, a Actor, sessionID uuid.UUID, in CloseCashSessionInput) (_ CashSessionView, err error) {
	ctx, span := s.startSpan(ctx, "CloseCashSession", a, cashSessionAttr(sessionID))
	defer func() { usecase.End(span, err) }()

	count, err := domain.NewCashCount(in.Counts)
	if err != nil {
		return CashSessionView{}, err
	}
	session, err := s.CashSessions.Close(ctx, a.TenantID, sessionID, func(session *domain.CashSession, activity domain.CashActivity) error {
		return session.Close(domain.CloseCashInput{
			UserID: a.UserID, Count: count, Activity: activity, Notes: in.Notes, At: s.Now(),
		})
	})
	if err != nil {
		return CashSessionView{}, err
	}
	closing := session.Closing()
	result := attribute.String("cash.result", closing.Result().String())
	span.SetAttributes(result)
	s.cashClosed.Add(ctx, 1, metric.WithAttributes(result))
	s.cashDifference.Record(ctx, closing.Difference.Abs().InexactFloat64(), metric.WithAttributes(result))
	s.Logger.InfoContext(ctx, "cash session closed",
		slog.String("cash_session_id", sessionID.String()), slog.String("result", closing.Result().String()))

	movements, err := s.CashSessions.Movements(ctx, a.TenantID, sessionID)
	if err != nil {
		return CashSessionView{}, err
	}
	return CashSessionView{Session: session, Movements: movements}, nil
}

// GetCashSession returns a session with its movements.
func (s *Service) GetCashSession(ctx context.Context, a Actor, id uuid.UUID) (_ CashSessionView, err error) {
	ctx, span := s.startSpan(ctx, "GetCashSession", a, cashSessionAttr(id))
	defer func() { usecase.End(span, err) }()

	session, err := s.CashSessions.Get(ctx, a.TenantID, id)
	if err != nil {
		return CashSessionView{}, err
	}
	return s.view(ctx, a, session)
}

// GetCurrentCashSession returns whether the tenant requires an open session
// to sell and the actor's open session with its movements, if any.
func (s *Service) GetCurrentCashSession(ctx context.Context, a Actor) (_ CurrentCashSession, err error) {
	ctx, span := s.startSpan(ctx, "GetCurrentCashSession", a)
	defer func() { usecase.End(span, err) }()

	policy, err := s.Policies.Get(ctx, a.TenantID)
	if err != nil {
		return CurrentCashSession{}, err
	}
	current := CurrentCashSession{Required: !policy.CashSessionOptional}
	session, err := s.CashSessions.Current(ctx, a.TenantID, a.UserID)
	switch {
	case errors.Is(err, domain.ErrCashSessionNotFound):
		return current, nil
	case err != nil:
		return CurrentCashSession{}, err
	}
	span.SetAttributes(cashSessionAttr(session.ID()))
	view, err := s.view(ctx, a, session)
	if err != nil {
		return CurrentCashSession{}, err
	}
	current.Session = &view
	return current, nil
}

func (s *Service) view(ctx context.Context, a Actor, session *domain.CashSession) (CashSessionView, error) {
	movements, err := s.CashSessions.Movements(ctx, a.TenantID, session.ID())
	if err != nil {
		return CashSessionView{}, err
	}
	return CashSessionView{Session: session, Movements: movements}, nil
}
