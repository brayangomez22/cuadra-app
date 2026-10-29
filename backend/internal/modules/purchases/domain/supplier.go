package domain

import (
	"time"

	"github.com/google/uuid"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/taxid"
)

const (
	maxSupplierNameLength = 200
	// maxPaymentTermDays is a year: a longer term is a loan, not a purchase.
	maxPaymentTermDays = 365
)

// SupplierParams are the data of a supplier, as the user writes them.
// PaymentTermDays is how many days after the invoice its purchases on
// credit fall due. Active is false only to deactivate.
type SupplierParams struct {
	Name            string
	NIT             string
	Contact         Contact
	PaymentTermDays int
	Active          bool
}

// Supplier is someone the tenant buys from, identified by its NIT (a person
// with a RUT has one too). A supplier is deactivated, never deleted,
// because its purchases name it.
type Supplier struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	createdAt time.Time
	supplierData
}

// supplierData is what Update replaces, already checked.
type supplierData struct {
	name     string
	nit      taxid.NIT
	contact  Contact
	termDays int
	active   bool
}

// NewSupplier creates a supplier with its own UUID v7.
func NewSupplier(tenantID uuid.UUID, p SupplierParams, now time.Time) (*Supplier, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return newSupplier(id, tenantID, p, now)
}

func newSupplier(id, tenantID uuid.UUID, p SupplierParams, createdAt time.Time) (*Supplier, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidSupplierID
	}
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	s := &Supplier{id: id, tenantID: tenantID, createdAt: createdAt.UTC()}
	if err := s.Update(p); err != nil {
		return nil, err
	}
	return s, nil
}

// Update replaces the supplier's data; on error the supplier is unchanged.
func (s *Supplier) Update(p SupplierParams) error {
	d, err := parseSupplierData(p)
	if err != nil {
		return err
	}
	s.supplierData = d
	return nil
}

func parseSupplierData(p SupplierParams) (supplierData, error) {
	var err error
	d := supplierData{termDays: p.PaymentTermDays, active: p.Active}
	if d.name, err = requiredText(p.Name, maxSupplierNameLength, ErrInvalidSupplierName); err != nil {
		return supplierData{}, err
	}
	if d.nit, err = taxid.ParseNIT(p.NIT); err != nil {
		return supplierData{}, err
	}
	if d.contact, err = p.Contact.normalize(); err != nil {
		return supplierData{}, err
	}
	if p.PaymentTermDays < 0 || p.PaymentTermDays > maxPaymentTermDays {
		return supplierData{}, ErrInvalidPaymentTerm
	}
	return d, nil
}

// dueAfter returns the day a purchase on credit invoiced on issued falls due
// under the supplier's term.
func (s *Supplier) dueAfter(issued Date) Date { return issued.AddDays(s.termDays) }

// ID returns the supplier ID (UUID v7).
func (s *Supplier) ID() uuid.UUID { return s.id }

// TenantID returns the tenant the supplier belongs to.
func (s *Supplier) TenantID() uuid.UUID { return s.tenantID }

// Name returns the supplier's legal or trade name.
func (s *Supplier) Name() string { return s.name }

// NIT returns the supplier's tax ID.
func (s *Supplier) NIT() taxid.NIT { return s.nit }

// Contact returns who to call at the supplier and how.
func (s *Supplier) Contact() Contact { return s.contact }

// PaymentTermDays returns the days of credit the supplier gives.
func (s *Supplier) PaymentTermDays() int { return s.termDays }

// Active reports whether the tenant still buys from the supplier.
func (s *Supplier) Active() bool { return s.active }

// CreatedAt returns the creation time in UTC.
func (s *Supplier) CreatedAt() time.Time { return s.createdAt }

// SupplierSnapshot holds a supplier's fields as they are stored.
type SupplierSnapshot struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	Name            string
	NIT             string
	Contact         Contact
	PaymentTermDays int
	Active          bool
	CreatedAt       time.Time
}

// Snapshot returns the supplier's fields, for the repository.
func (s *Supplier) Snapshot() SupplierSnapshot {
	return SupplierSnapshot{
		ID: s.id, TenantID: s.tenantID, Name: s.name, NIT: s.nit.String(), Contact: s.contact,
		PaymentTermDays: s.termDays, Active: s.active, CreatedAt: s.createdAt,
	}
}

// RehydrateSupplier rebuilds a stored supplier, checking the same
// invariants as NewSupplier.
func RehydrateSupplier(s SupplierSnapshot) (*Supplier, error) {
	return newSupplier(s.ID, s.TenantID, SupplierParams{
		Name: s.Name, NIT: s.NIT, Contact: s.Contact, PaymentTermDays: s.PaymentTermDays, Active: s.Active,
	}, s.CreatedAt)
}
