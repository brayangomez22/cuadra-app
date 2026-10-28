package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

const maxCustomerNameLength = 200

// PersonType tells a natural person from a company, as the electronic
// invoice reports the buyer.
type PersonType string

// Person types of a customer.
const (
	PersonNatural PersonType = "natural"
	PersonLegal   PersonType = "legal"
)

// CustomerParams are the data of a customer, as the user writes them.
// SegmentID is optional: nil means no segment. Active is false only to
// deactivate.
type CustomerParams struct {
	PersonType          PersonType
	DocumentType        DocumentType
	DocumentNumber      string
	Name                string
	TradeName           string
	Contact             Contact
	SegmentID           *uuid.UUID
	TaxResponsibilities []string
	Active              bool
}

// Customer is someone the tenant sells to, with what the electronic invoice
// needs of the buyer: person type, document and tax responsibilities. A
// customer is deactivated, never deleted, because its sales name it. A sale
// without a customer is to the final consumer: there is no stored customer
// for it.
type Customer struct {
	id        uuid.UUID
	tenantID  uuid.UUID
	createdAt time.Time
	customerData
}

// customerData is what Update replaces, already checked.
type customerData struct {
	personType PersonType
	document   Document
	name       string
	tradeName  string
	contact    Contact
	segmentID  *uuid.UUID
	taxes      []TaxResponsibility
	active     bool
}

// NewCustomer creates a customer with its own UUID v7.
func NewCustomer(tenantID uuid.UUID, p CustomerParams, now time.Time) (*Customer, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return newCustomer(id, tenantID, p, now)
}

func newCustomer(id, tenantID uuid.UUID, p CustomerParams, createdAt time.Time) (*Customer, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidCustomerID
	}
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}
	c := &Customer{id: id, tenantID: tenantID, createdAt: createdAt.UTC()}
	if err := c.Update(p); err != nil {
		return nil, err
	}
	return c, nil
}

// Update replaces the customer's data; on error the customer is unchanged.
func (c *Customer) Update(p CustomerParams) error {
	d, err := parseCustomerData(p)
	if err != nil {
		return err
	}
	c.customerData = d
	return nil
}

// parseCustomerData checks p: a known person type, a valid document (a
// company is identified by NIT), the segment and then the details.
func parseCustomerData(p CustomerParams) (customerData, error) {
	if p.PersonType != PersonNatural && p.PersonType != PersonLegal {
		return customerData{}, ErrInvalidPersonType
	}
	doc, err := ParseDocument(p.DocumentType, p.DocumentNumber)
	if err != nil {
		return customerData{}, err
	}
	if p.PersonType == PersonLegal && doc.Type() != DocumentNIT {
		return customerData{}, ErrLegalPersonRequiresNIT
	}
	if p.SegmentID != nil && *p.SegmentID == uuid.Nil {
		return customerData{}, ErrSegmentNotFound
	}
	d := customerData{personType: p.PersonType, document: doc, segmentID: copyID(p.SegmentID), active: p.Active}
	if err := d.parseDetails(p); err != nil {
		return customerData{}, err
	}
	return d, nil
}

// parseDetails checks the names, the contact and the tax responsibilities
// of p into d.
func (d *customerData) parseDetails(p CustomerParams) error {
	var err error
	if d.name, err = normalizeText(p.Name, maxCustomerNameLength, ErrInvalidCustomerName); err != nil {
		return err
	}
	if d.tradeName, err = optionalText(p.TradeName, maxCustomerNameLength, ErrInvalidCustomerName); err != nil {
		return err
	}
	if d.contact, err = p.Contact.normalize(); err != nil {
		return err
	}
	d.taxes, err = ParseTaxResponsibilities(p.TaxResponsibilities)
	return err
}

// ID returns the customer ID (UUID v7).
func (c *Customer) ID() uuid.UUID { return c.id }

// TenantID returns the tenant the customer belongs to.
func (c *Customer) TenantID() uuid.UUID { return c.tenantID }

// PersonType returns whether the customer is a person or a company.
func (c *Customer) PersonType() PersonType { return c.personType }

// Document returns the identification document.
func (c *Customer) Document() Document { return c.document }

// Name returns the full name or the company's legal name.
func (c *Customer) Name() string { return c.name }

// TradeName returns the trade name, or "" when it is the name.
func (c *Customer) TradeName() string { return c.tradeName }

// Contact returns how to reach the customer.
func (c *Customer) Contact() Contact { return c.contact }

// SegmentID returns a copy of the segment's id, or nil without segment.
func (c *Customer) SegmentID() *uuid.UUID { return copyID(c.segmentID) }

// TaxResponsibilities returns a copy of the tax responsibilities, in their
// fixed order.
func (c *Customer) TaxResponsibilities() []TaxResponsibility { return slices.Clone(c.taxes) }

// Active reports whether the customer can be sold to.
func (c *Customer) Active() bool { return c.active }

// CreatedAt returns the creation time in UTC.
func (c *Customer) CreatedAt() time.Time { return c.createdAt }
