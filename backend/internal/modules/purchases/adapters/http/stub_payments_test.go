package httpadapter_test

import (
	"context"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/app"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

func (s *stubUseCases) VoidPurchase(_ context.Context, a app.Actor, id uuid.UUID, reason string) (*domain.Purchase, error) {
	s.actor, s.id, s.reason = a, id, reason
	return s.purchase, s.err
}

func (s *stubUseCases) RecordSupplierPayment(_ context.Context, a app.Actor, id uuid.UUID, in app.SupplierPaymentRequest) (app.SupplierPaymentResult, error) {
	s.actor, s.id, s.payment = a, id, in
	return s.paid, s.err
}

func (s *stubUseCases) PurchasePayments(_ context.Context, a app.Actor, id uuid.UUID) ([]*domain.SupplierPayment, error) {
	s.actor, s.id = a, id
	return s.payments, s.err
}

func (s *stubUseCases) ListPayables(_ context.Context, a app.Actor, q app.PayablesQuery) (domain.PayablesPage, error) {
	s.actor, s.payablesQuery = a, q
	return s.payables, s.err
}
