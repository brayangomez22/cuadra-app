//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/adapters/postgres"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/dbtest"
)

// now has no sub-microsecond part, so it survives a round trip through timestamptz.
var now = time.Date(2026, 10, 28, 20, 30, 15, 123456000, time.UTC)

// Compile-time checks: the adapters satisfy the domain ports.
var (
	_ domain.CustomerRepository = (*postgres.CustomerRepository)(nil)
	_ domain.SegmentRepository  = (*postgres.SegmentRepository)(nil)
)

type fixture struct {
	db        *db.DB
	customers *postgres.CustomerRepository
	segments  *postgres.SegmentRepository
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	d := dbtest.New(t)
	return fixture{db: d, customers: postgres.NewCustomerRepository(d), segments: postgres.NewSegmentRepository(d)}
}

// createTenant stores a tenant row directly: customers does not depend on
// the identity module, but its tables reference tenants.
func (f fixture) createTenant(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	require.NoError(t, f.db.WithTenantTx(t.Context(), id, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(),
			"INSERT INTO tenants (id, name, nit, status, business_template, created_at) VALUES ($1, 'Negocio', '900123456-8', 'active', 'generica', $2)", id, now)
		return err
	}))
	return id
}

func (f fixture) createSegment(t *testing.T, tenantID uuid.UUID, name string) *domain.Segment {
	t.Helper()
	s, err := domain.NewSegment(tenantID, name, now)
	require.NoError(t, err)
	require.NoError(t, f.segments.Create(t.Context(), s))
	return s
}

func (f fixture) createCustomer(t *testing.T, tenantID uuid.UUID, p domain.CustomerParams) *domain.Customer {
	t.Helper()
	c, err := domain.NewCustomer(tenantID, p, now)
	require.NoError(t, err)
	require.NoError(t, f.customers.Create(t.Context(), c))
	return c
}

// person is a customer identified by cédula.
func person(document, name string) domain.CustomerParams {
	return domain.CustomerParams{
		PersonType: domain.PersonNatural, DocumentType: domain.DocumentCC, DocumentNumber: document, Name: name, Active: true,
	}
}

// company is a customer identified by NIT, with every optional field.
func company(nit, name string, segment *uuid.UUID) domain.CustomerParams {
	return domain.CustomerParams{
		PersonType: domain.PersonLegal, DocumentType: domain.DocumentNIT, DocumentNumber: nit, Name: name,
		TradeName: "La Moda", SegmentID: segment, TaxResponsibilities: []string{"O-13", "O-23"}, Active: true,
		Contact: domain.Contact{Email: "compras@lamoda.co", Phone: "604 444 5566", Address: "Cra 50 # 10-20", City: "Medellín"},
	}
}

func names(items []*domain.Customer) []string {
	out := make([]string, 0, len(items))
	for _, c := range items {
		out = append(out, c.Name())
	}
	return out
}
