package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/adapters/postgres/sqlcgen"
	"github.com/brayangomez22/cuadra-app/backend/internal/modules/customers/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/db"
)

// CustomerRepository implements domain.CustomerRepository.
type CustomerRepository struct {
	db *db.DB
}

// NewCustomerRepository returns a CustomerRepository on d.
func NewCustomerRepository(d *db.DB) *CustomerRepository { return &CustomerRepository{db: d} }

// Create stores a new customer.
func (r *CustomerRepository) Create(ctx context.Context, c *domain.Customer) error {
	s := c.Snapshot()
	err := r.db.WithTenantTx(ctx, s.TenantID, func(tx pgx.Tx) error {
		return sqlcgen.New(tx).CreateCustomer(ctx, sqlcgen.CreateCustomerParams{
			ID: s.ID, TenantID: s.TenantID, PersonType: string(s.PersonType), DocumentType: string(s.DocumentType),
			DocumentNumber: s.DocumentNumber, Name: s.Name, TradeName: nullable(s.TradeName),
			Email: nullable(s.Contact.Email), Phone: nullable(s.Contact.Phone), Address: nullable(s.Contact.Address),
			City: nullable(s.Contact.City), SegmentID: s.SegmentID, TaxResponsibilities: s.TaxResponsibilities,
			Active: s.Active, CreatedAt: s.CreatedAt,
		})
	})
	return customerWriteError("create", c, err)
}

// Update stores the customer's data; updated_at is the database's clock.
func (r *CustomerRepository) Update(ctx context.Context, c *domain.Customer) error {
	s := c.Snapshot()
	affected, err := db.Affected(ctx, r.db, s.TenantID, func(tx pgx.Tx) (int64, error) {
		return sqlcgen.New(tx).UpdateCustomer(ctx, sqlcgen.UpdateCustomerParams{
			ID: s.ID, PersonType: string(s.PersonType), DocumentType: string(s.DocumentType),
			DocumentNumber: s.DocumentNumber, Name: s.Name, TradeName: nullable(s.TradeName),
			Email: nullable(s.Contact.Email), Phone: nullable(s.Contact.Phone), Address: nullable(s.Contact.Address),
			City: nullable(s.Contact.City), SegmentID: s.SegmentID, TaxResponsibilities: s.TaxResponsibilities,
			Active: s.Active,
		})
	})
	if err == nil && affected == 0 {
		return domain.ErrCustomerNotFound
	}
	return customerWriteError("update", c, err)
}

// customerWriteError translates the constraints a write can violate: the
// unique document and the segment of another tenant (or none).
func customerWriteError(op string, c *domain.Customer, err error) error {
	switch {
	case db.Violates(err, db.UniqueViolation, customerDocKey):
		return &domain.DocumentTakenError{Document: c.Document()}
	case db.Violates(err, db.ForeignKeyViolation, customerSegmentKey):
		return domain.ErrSegmentNotFound
	case err != nil:
		return fmt.Errorf("customers: %s customer: %w", op, err)
	}
	return nil
}

// GetByID returns domain.ErrCustomerNotFound when the customer does not
// exist in tenantID.
func (r *CustomerRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	row, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) (sqlcgen.GetCustomerRow, error) {
		return sqlcgen.New(tx).GetCustomer(ctx, id)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCustomerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("customers: get customer: %w", err)
	}
	return customerFromRow(customerRow(row))
}

// customerRow is a stored customer, as every query reads it.
type customerRow sqlcgen.GetCustomerRow

func customerFromRow(row customerRow) (*domain.Customer, error) {
	c, err := domain.RehydrateCustomer(domain.CustomerSnapshot{
		ID: row.ID, TenantID: row.TenantID, PersonType: domain.PersonType(row.PersonType),
		DocumentType: domain.DocumentType(row.DocumentType), DocumentNumber: row.DocumentNumber,
		Name: row.Name, TradeName: value(row.TradeName),
		Contact: domain.Contact{
			Email: value(row.Email), Phone: value(row.Phone), Address: value(row.Address), City: value(row.City),
		},
		SegmentID: row.SegmentID, TaxResponsibilities: row.TaxResponsibilities, Active: row.Active,
		CreatedAt: row.CreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("customers: customer %s: %w", row.ID, err)
	}
	return c, nil
}

// nullable stores "" as NULL.
func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func value(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Names returns the names of the tenant's customers among ids.
func (r *CustomerRepository) Names(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	rows, err := db.Read(ctx, r.db, tenantID, func(tx pgx.Tx) ([]sqlcgen.CustomerNamesRow, error) {
		return sqlcgen.New(tx).CustomerNames(ctx, ids)
	})
	if err != nil {
		return nil, fmt.Errorf("customers: names: %w", err)
	}
	out := make(map[uuid.UUID]string, len(rows))
	for _, row := range rows {
		out[row.ID] = row.Name
	}
	return out, nil
}
