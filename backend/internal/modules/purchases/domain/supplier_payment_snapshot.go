package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// SupplierPaymentSnapshot holds a payment's fields as they are stored.
type SupplierPaymentSnapshot struct {
	ID, TenantID, PurchaseID, SupplierID uuid.UUID
	Method                               PaymentMethod
	Amount, BalanceAfter                 decimal.Decimal
	PaidOn                               Date
	UserID                               uuid.UUID
	CreatedAt                            time.Time
}

// Snapshot returns the payment's fields, for the repository.
func (s *SupplierPayment) Snapshot() SupplierPaymentSnapshot {
	return SupplierPaymentSnapshot{
		ID: s.id, TenantID: s.tenantID, PurchaseID: s.purchaseID, SupplierID: s.supplierID, Method: s.method,
		Amount: s.amount, BalanceAfter: s.balanceAfter, PaidOn: s.paidOn, UserID: s.userID, CreatedAt: s.createdAt,
	}
}

// RehydrateSupplierPayment rebuilds a stored payment, checking its ids, its
// method and that it paid something without leaving a negative balance.
func RehydrateSupplierPayment(s SupplierPaymentSnapshot) (*SupplierPayment, error) {
	if s.ID == uuid.Nil || s.TenantID == uuid.Nil || s.PurchaseID == uuid.Nil || s.SupplierID == uuid.Nil ||
		s.UserID == uuid.Nil || !s.Method.valid() || !s.Amount.IsPositive() || s.BalanceAfter.IsNegative() ||
		s.PaidOn.IsZero() {
		return nil, ErrInvalidStoredPayment
	}
	return &SupplierPayment{
		id: s.ID, tenantID: s.TenantID, purchaseID: s.PurchaseID, supplierID: s.SupplierID, method: s.Method,
		amount: s.Amount, balanceAfter: s.BalanceAfter, paidOn: s.PaidOn, userID: s.UserID, createdAt: s.CreatedAt.UTC(),
	}, nil
}
