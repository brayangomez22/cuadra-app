package app

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/sales/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ReceiveCash records cash the actor received outside a sale (a customer's
// payment, for the receivables module) as a cash in of their open session,
// for reason, and returns the session's id. Without an open session it
// returns uuid.Nil if the tenant lets users work without one, and
// domain.ErrCashSessionRequired otherwise. With ctx's transaction, the
// movement joins it.
func (s *Service) ReceiveCash(ctx context.Context, a Actor, amount decimal.Decimal, reason string) (_ uuid.UUID, err error) {
	ctx, span := s.startSpan(ctx, "ReceiveCash", a)
	defer func() { usecase.End(span, err) }()

	policy, err := s.Policies.Get(ctx, a.TenantID)
	if err != nil {
		return uuid.Nil, err
	}
	session, err := s.CashSessions.Current(ctx, a.TenantID, a.UserID)
	switch {
	case errors.Is(err, domain.ErrCashSessionNotFound) && policy.CashSessionOptional:
		return uuid.Nil, nil
	case errors.Is(err, domain.ErrCashSessionNotFound):
		return uuid.Nil, domain.ErrCashSessionRequired
	case err != nil:
		return uuid.Nil, err
	}
	span.SetAttributes(cashSessionAttr(session.ID()))
	if _, err := s.RecordCashMovement(ctx, a, session.ID(), CashMovementRequest{
		Kind: string(domain.CashIn), Amount: amount, Reason: reason,
	}); err != nil {
		return uuid.Nil, err
	}
	return session.ID(), nil
}
