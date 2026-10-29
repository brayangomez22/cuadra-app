package app

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// PaymentRequest is a payment of a customer; Method is a
// domain.PaymentMethod code.
type PaymentRequest struct {
	Method string
	Amount decimal.Decimal
}

// PaymentResult is the payment received and the customer's account after
// it.
type PaymentResult struct {
	Payment domain.Payment
	Account *domain.Account
}

// cashReason is the reason of the cash in a cash payment records in the
// receiver's drawer.
const cashReason = "Abono de cartera"

// RecordPayment receives a payment from a customer (active or not) and
// applies it to what it owes, the receivable due first first. It cannot
// exceed the balance (domain.ErrPaymentExceedsBalance). A cash payment goes
// into the actor's open cash session in the same transaction, so the drawer
// adds up at closing.
func (s *Service) RecordPayment(ctx context.Context, a Actor, customerID uuid.UUID, in PaymentRequest) (_ PaymentResult, err error) {
	ctx, span := s.startSpan(ctx, "RecordPayment", a, customerAttr(customerID))
	defer func() { usecase.End(span, err) }()

	method, err := domain.ParsePaymentMethod(in.Method)
	if err != nil {
		return PaymentResult{}, err
	}
	if err := s.Customers.CheckCustomer(ctx, a.TenantID, a.UserID, customerID); err != nil {
		return PaymentResult{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return PaymentResult{}, err
	}
	span.SetAttributes(attribute.String("payment.id", id.String()))
	payment := domain.PaymentInput{ID: id, Method: method, Amount: in.Amount, UserID: a.UserID, At: s.Now()}
	var res PaymentResult
	err = s.Credit.Update(ctx, a.TenantID, customerID, func(ctx context.Context, account *domain.Account, open []*domain.Receivable) (domain.Change, error) {
		if err := s.toDrawer(ctx, a, &payment); err != nil {
			return domain.Change{}, err
		}
		p, err := account.Collect(payment, open)
		if err != nil {
			return domain.Change{}, err
		}
		res = PaymentResult{Payment: p, Account: account}
		return domain.Change{Payment: &p, Changed: open}, nil
	})
	if err != nil {
		return PaymentResult{}, err
	}
	methodAttr := attribute.String("payment.method", method.String())
	s.payments.Add(ctx, 1, metric.WithAttributes(methodAttr))
	s.paymentAmount.Record(ctx, in.Amount.InexactFloat64(), metric.WithAttributes(methodAttr))
	s.Logger.InfoContext(ctx, "customer payment received",
		slog.String("payment_id", id.String()), slog.String("customer_id", customerID.String()))
	return res, nil
}

// toDrawer puts a cash payment in the actor's open cash session and notes
// the session in it. Other methods do not touch the drawer.
func (s *Service) toDrawer(ctx context.Context, a Actor, p *domain.PaymentInput) error {
	if p.Method != domain.PaymentCash {
		return nil
	}
	sessionID, err := s.Cash.ReceiveCash(ctx, a.TenantID, a.UserID, p.Amount, cashReason)
	if err != nil {
		return err
	}
	if sessionID != uuid.Nil {
		p.CashSessionID = &sessionID
	}
	return nil
}
