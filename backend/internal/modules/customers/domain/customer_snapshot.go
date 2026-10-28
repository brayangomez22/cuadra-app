package domain

import (
	"time"

	"github.com/google/uuid"
)

// CustomerSnapshot holds a customer's fields as they are stored.
type CustomerSnapshot struct {
	ID                  uuid.UUID
	TenantID            uuid.UUID
	PersonType          PersonType
	DocumentType        DocumentType
	DocumentNumber      string
	Name                string
	TradeName           string
	Contact             Contact
	SegmentID           *uuid.UUID
	TaxResponsibilities []string
	Active              bool
	CreatedAt           time.Time
}

// Snapshot returns the customer's fields, for the repository.
func (c *Customer) Snapshot() CustomerSnapshot {
	taxes := make([]string, len(c.taxes))
	for i, r := range c.taxes {
		taxes[i] = string(r)
	}
	return CustomerSnapshot{
		ID: c.id, TenantID: c.tenantID, PersonType: c.personType,
		DocumentType: c.document.Type(), DocumentNumber: c.document.Number(),
		Name: c.name, TradeName: c.tradeName, Contact: c.contact, SegmentID: copyID(c.segmentID),
		TaxResponsibilities: taxes, Active: c.active, CreatedAt: c.createdAt,
	}
}

// RehydrateCustomer rebuilds a stored customer, checking the same invariants
// as NewCustomer so corrupt data surfaces as an error instead of an invalid
// entity.
func RehydrateCustomer(s CustomerSnapshot) (*Customer, error) {
	return newCustomer(s.ID, s.TenantID, CustomerParams{
		PersonType: s.PersonType, DocumentType: s.DocumentType, DocumentNumber: s.DocumentNumber,
		Name: s.Name, TradeName: s.TradeName, Contact: s.Contact, SegmentID: s.SegmentID,
		TaxResponsibilities: s.TaxResponsibilities, Active: s.Active,
	}, s.CreatedAt)
}
