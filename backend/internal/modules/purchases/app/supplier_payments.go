package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// SupplierPaymentRequest is a payment to the supplier of a purchase. Method
// is a domain.PaymentMethod code; PaidOn defaults to today in
// America/Bogota.
type SupplierPaymentRequest struct {
	Method string
	Amount decimal.Decimal
	PaidOn *domain.Date
}

// SupplierPaymentResult is the payment recorded and the purchase after it.
type SupplierPaymentResult struct {
	Payment  *domain.SupplierPayment
	Purchase *domain.Purchase
}

// PayablesQuery is a page of what the tenant owes its suppliers; the HTTP
// adapter bounds Limit and Offset.
type PayablesQuery struct {
	Limit  int
	Offset int
}

// RecordSupplierPayment records a payment to the supplier of a purchase on
// credit and lowers what is left to pay, never below zero. It does not touch
// the cash drawer: money taken from it is recorded there as an outflow.
func (s *Service) RecordSupplierPayment(ctx context.Context, a Actor, purchaseID uuid.UUID, in SupplierPaymentRequest) (_ SupplierPaymentResult, err error) {
	ctx, span := s.startSpan(ctx, "RecordSupplierPayment", a, purchaseAttr(purchaseID))
	defer func() { usecase.End(span, err) }()

	method, err := domain.ParsePaymentMethod(in.Method)
	if err != nil {
		return SupplierPaymentResult{}, err
	}
	now := s.Now()
	paidOn := domain.DateIn(now)
	if in.PaidOn != nil {
		paidOn = *in.PaidOn
	}
	p, payment, err := s.Purchases.Pay(ctx, a.TenantID, purchaseID, func(p *domain.Purchase) (*domain.SupplierPayment, error) {
		return p.Pay(domain.PaymentInput{Method: method, Amount: in.Amount, PaidOn: paidOn, UserID: a.UserID}, now)
	})
	if err != nil {
		return SupplierPaymentResult{}, err
	}
	span.SetAttributes(attribute.String("payment.id", payment.ID().String()))
	s.supplierPayments.Add(ctx, 1, metric.WithAttributes(attribute.String("method", string(method))))
	return SupplierPaymentResult{Payment: payment, Purchase: p}, nil
}

// PurchasePayments returns the payments of a purchase of the tenant, the
// first one first.
func (s *Service) PurchasePayments(ctx context.Context, a Actor, purchaseID uuid.UUID) (_ []*domain.SupplierPayment, err error) {
	ctx, span := s.startSpan(ctx, "PurchasePayments", a, purchaseAttr(purchaseID))
	defer func() { usecase.End(span, err) }()

	if _, err := s.Purchases.GetByID(ctx, a.TenantID, purchaseID); err != nil {
		return nil, err
	}
	return s.Purchases.Payments(ctx, a.TenantID, purchaseID)
}

// ListPayables returns what the tenant owes each supplier, split into what
// is overdue and what falls due within a week of today in America/Bogota.
func (s *Service) ListPayables(ctx context.Context, a Actor, q PayablesQuery) (_ domain.PayablesPage, err error) {
	ctx, span := s.startSpan(ctx, "ListPayables", a)
	defer func() { usecase.End(span, err) }()

	w := domain.PayablesWindowOn(domain.DateIn(s.Now()))
	return s.Purchases.Payables(ctx, a.TenantID, w, q.Limit, q.Offset)
}
