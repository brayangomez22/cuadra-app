package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
)

// change locks the tenant's purchase, loads it and calls fn in one
// transaction, which joins ctx so the inventory runs in it. fn's own error
// (a rule of the domain, the inventory's) comes back as is.
func (r *PurchaseRepository) change(ctx context.Context, tenantID, id uuid.UUID, fn func(context.Context, *sqlcgen.Queries, *domain.Purchase) error) (*domain.Purchase, error) {
	var p *domain.Purchase
	var fnErr error
	err := r.db.InTenantTx(ctx, tenantID, func(ctx context.Context, tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if _, err := q.LockPurchase(ctx, id); err != nil {
			return err
		}
		var err error
		if p, err = loadPurchase(ctx, q, id); err != nil {
			return err
		}
		fnErr = fn(ctx, q, p)
		return fnErr
	})
	switch {
	case fnErr != nil:
		return nil, fnErr
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, domain.ErrPurchaseNotFound):
		return nil, domain.ErrPurchaseNotFound
	case err != nil:
		return nil, fmt.Errorf("purchases: change purchase %s: %w", id, err)
	}
	return p, nil
}

// Void locks the purchase, calls void (which takes out its stock through
// ctx) and stores it voided, all in one transaction.
func (r *PurchaseRepository) Void(ctx context.Context, tenantID, id uuid.UUID, void func(context.Context, *domain.Purchase) error) (*domain.Purchase, error) {
	return r.change(ctx, tenantID, id, func(ctx context.Context, q *sqlcgen.Queries, p *domain.Purchase) error {
		if err := void(ctx, p); err != nil {
			return err
		}
		v := p.Voided()
		if v == nil {
			return errors.New("purchases: void did not void the purchase")
		}
		return q.UpdatePurchaseVoid(ctx, sqlcgen.UpdatePurchaseVoidParams{
			VoidedAt: &v.At, VoidedBy: &v.By, VoidReason: &v.Reason, BalanceDue: p.BalanceDue(), ID: p.ID(),
		})
	})
}

// Pay locks the purchase, calls pay and stores the new balance and the
// payment in one transaction.
func (r *PurchaseRepository) Pay(ctx context.Context, tenantID, id uuid.UUID, pay func(*domain.Purchase) (*domain.SupplierPayment, error)) (*domain.Purchase, *domain.SupplierPayment, error) {
	var payment *domain.SupplierPayment
	p, err := r.change(ctx, tenantID, id, func(ctx context.Context, q *sqlcgen.Queries, p *domain.Purchase) error {
		var err error
		if payment, err = pay(p); err != nil {
			return err
		}
		if err := q.UpdatePurchaseBalance(ctx, sqlcgen.UpdatePurchaseBalanceParams{BalanceDue: p.BalanceDue(), ID: p.ID()}); err != nil {
			return err
		}
		s := payment.Snapshot()
		return q.InsertSupplierPayment(ctx, sqlcgen.InsertSupplierPaymentParams{
			ID: s.ID, TenantID: s.TenantID, PurchaseID: s.PurchaseID, SupplierID: s.SupplierID, Method: string(s.Method),
			Amount: s.Amount, BalanceAfter: s.BalanceAfter, PaidOn: s.PaidOn.Time(), CreatedBy: s.UserID,
			CreatedAt: s.CreatedAt,
		})
	})
	if err != nil {
		return nil, nil, err
	}
	return p, payment, nil
}
