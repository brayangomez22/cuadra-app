package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// TermsRequest is a customer's credit limit (COP) and term (days).
type TermsRequest struct {
	Limit decimal.Decimal
	Days  int
}

// CreditView is a customer's credit account and its open receivables, due
// first first.
type CreditView struct {
	Account *domain.Account
	Open    []*domain.Receivable
}

// CustomerCredit returns the credit of a customer of the tenant.
func (s *Service) CustomerCredit(ctx context.Context, a Actor, customerID uuid.UUID) (_ CreditView, err error) {
	ctx, span := s.startSpan(ctx, "CustomerCredit", a, customerAttr(customerID))
	defer func() { usecase.End(span, err) }()

	if err := s.Customers.CheckCustomer(ctx, a.TenantID, a.UserID, customerID); err != nil {
		return CreditView{}, err
	}
	account, err := s.Credit.Account(ctx, a.TenantID, customerID)
	if err != nil {
		return CreditView{}, err
	}
	open, err := s.Credit.Open(ctx, a.TenantID, customerID)
	if err != nil {
		return CreditView{}, err
	}
	return CreditView{Account: account, Open: open}, nil
}

// SetCreditTerms sets a customer's limit and term (zero limit: no credit).
// The balance and the due dates of what it owes do not change.
func (s *Service) SetCreditTerms(ctx context.Context, a Actor, customerID uuid.UUID, in TermsRequest) (_ *domain.Account, err error) {
	ctx, span := s.startSpan(ctx, "SetCreditTerms", a, customerAttr(customerID))
	defer func() { usecase.End(span, err) }()

	terms, err := domain.NewTerms(in.Limit, in.Days)
	if err != nil {
		return nil, err
	}
	if err := s.Customers.CheckCustomer(ctx, a.TenantID, a.UserID, customerID); err != nil {
		return nil, err
	}
	account, err := s.Credit.Account(ctx, a.TenantID, customerID)
	if err != nil {
		return nil, err
	}
	account.SetTerms(terms)
	if err := s.Credit.SaveTerms(ctx, account, s.Now()); err != nil {
		return nil, err
	}
	return account, nil
}
