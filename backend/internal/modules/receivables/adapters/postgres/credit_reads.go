package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// OfSale returns the receivable saleID opened.
func (r *CreditRepository) OfSale(ctx context.Context, tenantID, saleID uuid.UUID) (domain.SaleReceivable, error) {
	row, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (sqlcgen.ReceivableOfSaleRow, error) {
		return sqlcgen.New(tx).ReceivableOfSale(ctx, saleID)
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.SaleReceivable{}, domain.ErrReceivableNotFound
	case err != nil:
		return domain.SaleReceivable{}, fmt.Errorf("receivables: receivable of sale: %w", err)
	}
	return domain.SaleReceivable{ID: row.ID, CustomerID: row.CustomerID, Status: domain.ReceivableStatus(row.Status)}, nil
}

// Open returns the customer's open receivables, due first first.
func (r *CreditRepository) Open(ctx context.Context, tenantID, customerID uuid.UUID) ([]*domain.Receivable, error) {
	rows, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]sqlcgen.Receivable, error) {
		return sqlcgen.New(tx).OpenReceivables(ctx, customerID)
	})
	if err != nil {
		return nil, fmt.Errorf("receivables: open receivables: %w", err)
	}
	return receivablesOf(rows), nil
}

// customerActivity is everything that moved a customer's balance.
type customerActivity struct {
	receivables []sqlcgen.Receivable
	payments    []sqlcgen.CustomerPaymentsRow
}

// Movements returns the customer's credit sales, cancellations and
// payments, unordered and without balances.
func (r *CreditRepository) Movements(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Movement, error) {
	act, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (customerActivity, error) {
		q := sqlcgen.New(tx)
		receivables, err := q.CustomerReceivables(ctx, customerID)
		if err != nil {
			return customerActivity{}, err
		}
		payments, err := q.CustomerPayments(ctx, customerID)
		return customerActivity{receivables: receivables, payments: payments}, err
	})
	if err != nil {
		return nil, fmt.Errorf("receivables: movements: %w", err)
	}
	out := make([]domain.Movement, 0, len(act.receivables)+len(act.payments))
	for _, row := range act.receivables {
		saleID, due := row.SaleID, domain.DateOf(row.DueDate)
		out = append(out, domain.Movement{Kind: domain.MovementCharge, At: row.CreatedAt, Amount: row.Amount, SaleID: &saleID, DueDate: &due})
		if row.CanceledAt != nil {
			out = append(out, domain.Movement{Kind: domain.MovementCancellation, At: *row.CanceledAt, Amount: row.Amount, SaleID: &saleID})
		}
	}
	for _, row := range act.payments {
		id := row.ID
		out = append(out, domain.Movement{
			Kind: domain.MovementPayment, At: row.CreatedAt, Amount: row.Amount, PaymentID: &id, Method: domain.PaymentMethod(row.Method),
		})
	}
	return out, nil
}

// OpenBalances returns the balance and due date of every open receivable
// of the tenant.
func (r *CreditRepository) OpenBalances(ctx context.Context, tenantID uuid.UUID) ([]domain.OpenBalance, error) {
	rows, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]sqlcgen.OpenBalancesRow, error) {
		return sqlcgen.New(tx).OpenBalances(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("receivables: open balances: %w", err)
	}
	out := make([]domain.OpenBalance, len(rows))
	for i, row := range rows {
		out[i] = domain.OpenBalance{CustomerID: row.CustomerID, DueDate: domain.DateOf(row.DueDate), Balance: row.Balance}
	}
	return out, nil
}

func noAccount(tenantID, customerID uuid.UUID) *domain.Account {
	return domain.RehydrateAccount(domain.AccountSnapshot{TenantID: tenantID, CustomerID: customerID, Balance: decimal.Zero})
}

func accountOf(row sqlcgen.CreditAccount) *domain.Account {
	return domain.RehydrateAccount(domain.AccountSnapshot{
		TenantID: row.TenantID, CustomerID: row.CustomerID, Limit: row.CreditLimit,
		TermDays: int(row.TermDays), Balance: row.Balance,
	})
}

func receivablesOf(rows []sqlcgen.Receivable) []*domain.Receivable {
	out := make([]*domain.Receivable, len(rows))
	for i, row := range rows {
		s := domain.ReceivableSnapshot{
			ID: row.ID, TenantID: row.TenantID, CustomerID: row.CustomerID, SaleID: row.SaleID,
			Amount: row.Amount, Balance: row.Balance, DueDate: domain.DateOf(row.DueDate),
			Status: domain.ReceivableStatus(row.Status), CreatedAt: row.CreatedAt, CanceledAt: row.CanceledAt,
		}
		if row.AuthorizedBy != nil && row.AuthorizedAt != nil {
			s.Authorization = &domain.Authorization{UserID: *row.AuthorizedBy, At: *row.AuthorizedAt}
		}
		out[i] = domain.RehydrateReceivable(s)
	}
	return out
}
