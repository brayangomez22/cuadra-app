package app

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ChargeRequest is a credit sale: the customer, the sale, the amount sold
// on credit and, if the sale may exceed the available credit, who
// authorized it.
type ChargeRequest struct {
	CustomerID   uuid.UUID
	SaleID       uuid.UUID
	Amount       decimal.Decimal
	AuthorizedBy *uuid.UUID
}

// Charge opens the receivable of a credit sale, made by the actor, at the
// current time. The sales module calls it within the sale's transaction
// (ctx carries it), so the sale and its receivable are stored together; the
// customer's account stays locked until the sale commits, so two sales
// cannot use the same available credit. It fails with
// domain.ErrCreditNotEnabled if the customer has no credit and with a
// *domain.CreditLimitExceededError if the amount exceeds the available
// credit and nobody authorized it.
func (s *Service) Charge(ctx context.Context, a Actor, in ChargeRequest) (_ *domain.Receivable, err error) {
	ctx, span := s.startSpan(ctx, "Charge", a, customerAttr(in.CustomerID), saleAttr(in.SaleID))
	defer func() { usecase.End(span, err) }()

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	now := s.Now()
	charge := domain.ChargeInput{ID: id, SaleID: in.SaleID, Amount: in.Amount, At: now}
	if in.AuthorizedBy != nil {
		charge.Authorization = &domain.Authorization{UserID: *in.AuthorizedBy, At: now}
	}
	var opened *domain.Receivable
	err = s.Credit.Update(ctx, a.TenantID, in.CustomerID, func(_ context.Context, account *domain.Account, _ []*domain.Receivable) (domain.Change, error) {
		r, err := account.Charge(charge)
		if err != nil {
			return domain.Change{}, err
		}
		opened = r
		return domain.Change{Opened: r}, nil
	})
	if errors.Is(err, domain.ErrCreditLimitExceeded) {
		s.limitRejections.Add(ctx, 1)
	}
	if err != nil {
		return nil, err
	}
	authorized := attribute.Bool("credit.authorized", opened.Authorization() != nil)
	span.SetAttributes(attribute.String("receivable.id", id.String()), authorized)
	s.charges.Add(ctx, 1, metric.WithAttributes(authorized))
	return opened, nil
}

// CancelSale cancels the receivable of a voided sale, within the void's
// transaction (ctx carries it). It fails with domain.ErrReceivableNotFound
// if the sale opened none and with domain.ErrReceivableHasPayments if the
// customer already paid part of it.
func (s *Service) CancelSale(ctx context.Context, a Actor, saleID uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "CancelSale", a, saleAttr(saleID))
	defer func() { usecase.End(span, err) }()

	of, err := s.Credit.OfSale(ctx, a.TenantID, saleID)
	if err != nil {
		return err
	}
	span.SetAttributes(customerAttr(of.CustomerID), attribute.String("receivable.id", of.ID.String()))
	switch of.Status {
	case domain.ReceivablePaid:
		return domain.ErrReceivableHasPayments
	case domain.ReceivableCanceled:
		return domain.ErrReceivableNotOpen
	}
	err = s.Credit.Update(ctx, a.TenantID, of.CustomerID, func(_ context.Context, account *domain.Account, open []*domain.Receivable) (domain.Change, error) {
		i := slices.IndexFunc(open, func(r *domain.Receivable) bool { return r.ID() == of.ID })
		if i < 0 {
			return domain.Change{}, domain.ErrReceivableNotOpen
		}
		if err := account.Cancel(open[i], s.Now()); err != nil {
			return domain.Change{}, err
		}
		return domain.Change{Changed: open[i : i+1]}, nil
	})
	if err != nil {
		return err
	}
	s.cancellations.Add(ctx, 1)
	return nil
}
