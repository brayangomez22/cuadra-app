//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/receivables/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// now has no sub-microsecond part, so it survives a round trip through
// timestamptz.
var now = time.Date(2026, 10, 28, 20, 30, 15, 123456000, time.UTC)

// Compile-time check: the adapter satisfies the domain port.
var _ domain.CreditRepository = (*postgres.CreditRepository)(nil)

type fixture struct {
	db     *db.DB
	credit *postgres.CreditRepository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	d := dbtest.New(t)
	return fixture{db: d, credit: postgres.NewCreditRepository(d)}
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// tenant is a tenant with a location, for the sales its receivables
// reference.
type tenant struct {
	id, location uuid.UUID
	sales        int64
}

// createTenant stores a tenant and a location directly: receivables does
// not depend on identity nor inventory, but its tables reference them.
func (f fixture) createTenant(t *testing.T) *tenant {
	t.Helper()
	tn := &tenant{id: newID(), location: newID()}
	f.exec(t, tn.id, "INSERT INTO tenants (id, name, nit, status, business_template, created_at) VALUES ($1, 'Negocio', '900123456-8', 'active', 'generica', $2)", tn.id, now)
	f.exec(t, tn.id, "INSERT INTO locations (id, tenant_id, name, active, created_at) VALUES ($1, $2, 'Principal', true, $3)", tn.location, tn.id, now)
	return tn
}

// createCustomer stores a customer with a cédula.
func (f fixture) createCustomer(t *testing.T, tn *tenant, name string) uuid.UUID {
	t.Helper()
	id := newID()
	f.exec(t, tn.id, `INSERT INTO customers (id, tenant_id, person_type, document_type, document_number, name,
		tax_responsibilities, active, created_at, updated_at)
		VALUES ($1, $2, 'natural', 'cc', $3, $4, '{R-99-PN}', true, $5, $5)`, id, tn.id, id.String()[:12], name, now)
	return id
}

// createSale stores a confirmed sale of total amount, for a receivable to
// reference.
func (f fixture) createSale(t *testing.T, tn *tenant, amount string) uuid.UUID {
	t.Helper()
	id := newID()
	tn.sales++
	f.exec(t, tn.id, `INSERT INTO sales (id, tenant_id, location_id, number, user_id, status, subtotal, discounts, tax,
		total, paid, change, created_at, confirmed_at)
		VALUES ($1, $2, $3, $4, $5, 'confirmed', $6, 0, 0, $6, $6, 0, $7, $7)`, id, tn.id, tn.location, tn.sales, newID(), dec(amount), now)
	return id
}

func (f fixture) exec(t *testing.T, tenantID uuid.UUID, sql string, args ...any) {
	t.Helper()
	require.NoError(t, f.db.WithTenantTx(t.Context(), tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(), sql, args...)
		return err
	}))
}

// setTerms gives a customer a limit and a term.
func (f fixture) setTerms(t *testing.T, tn *tenant, customerID uuid.UUID, limit string, days int) {
	t.Helper()
	terms, err := domain.NewTerms(dec(limit), days)
	require.NoError(t, err)
	a, err := f.credit.Account(t.Context(), tn.id, customerID)
	require.NoError(t, err)
	a.SetTerms(terms)
	require.NoError(t, f.credit.SaveTerms(t.Context(), a, now))
}

// charge opens a receivable of a new sale, as the receivables use case does.
func (f fixture) charge(t *testing.T, tn *tenant, customerID uuid.UUID, amount string) *domain.Receivable {
	t.Helper()
	saleID := f.createSale(t, tn, amount)
	var opened *domain.Receivable
	err := f.credit.Update(t.Context(), tn.id, customerID, func(_ context.Context, a *domain.Account, _ []*domain.Receivable) (domain.Change, error) {
		r, err := a.Charge(domain.ChargeInput{ID: newID(), SaleID: saleID, Amount: dec(amount), At: now})
		opened = r
		return domain.Change{Opened: r}, err
	})
	require.NoError(t, err)
	return opened
}

// pay receives a payment by transfer, as the receivables use case does.
func (f fixture) pay(t *testing.T, tn *tenant, customerID uuid.UUID, amount string) error {
	t.Helper()
	return f.credit.Update(t.Context(), tn.id, customerID, func(_ context.Context, a *domain.Account, open []*domain.Receivable) (domain.Change, error) {
		p, err := a.Collect(domain.PaymentInput{
			ID: newID(), Method: domain.PaymentTransfer, Amount: dec(amount), UserID: newID(), At: now.Add(time.Hour),
		}, open)
		return domain.Change{Payment: &p, Changed: open}, err
	})
}

// openSum adds up the balances of the customer's open receivables.
func (f fixture) openSum(t *testing.T, tn *tenant, customerID uuid.UUID) decimal.Decimal {
	t.Helper()
	open, err := f.credit.Open(t.Context(), tn.id, customerID)
	require.NoError(t, err)
	sum := decimal.Zero
	for _, r := range open {
		sum = sum.Add(r.Balance())
	}
	return sum
}
