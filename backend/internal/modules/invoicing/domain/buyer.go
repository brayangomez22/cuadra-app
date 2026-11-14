package domain

import "github.com/google/uuid"

// The final consumer, who buys without identifying: the DIAN's generic
// document and name (P93).
const (
	FinalConsumerDocument = "222222222222"
	FinalConsumerName     = "Consumidor final"
	// FinalConsumerDocumentType is the type of document the final consumer
	// is sent with: a cédula, like the DIAN's generic buyer.
	FinalConsumerDocumentType = "cc"
	// finalConsumerIdentificationCode is the DIAN's code of the cédula.
	finalConsumerIdentificationCode = "13"
)

// Buyer is who the document is issued to: a customer of the tenant, or the
// final consumer.
type Buyer struct {
	// CustomerID is nil for the final consumer.
	CustomerID     *uuid.UUID
	PersonType     string
	DocumentType   string
	DocumentNumber string
	Name           string
	Email          string
	// IdentificationCode is the DIAN's code of DocumentType ("13", "31").
	IdentificationCode string
	Address            string
	// City is free text; MunicipalityCode, the DANE code ("05001"), empty
	// for a customer who has only the City.
	City             string
	MunicipalityCode string
}

// DepartmentCode is the DANE code of the buyer's department: the first two
// digits of its municipality's. Empty without a municipality.
func (b Buyer) DepartmentCode() string {
	const municipalityLength, departmentLength = 5, 2
	if len(b.MunicipalityCode) != municipalityLength {
		return ""
	}
	return b.MunicipalityCode[:departmentLength]
}

// FinalConsumer is the buyer of a sale without a customer.
func FinalConsumer() Buyer {
	return Buyer{
		PersonType: "natural", DocumentType: FinalConsumerDocumentType,
		IdentificationCode: finalConsumerIdentificationCode,
		DocumentNumber:     FinalConsumerDocument, Name: FinalConsumerName,
	}
}

// IsFinalConsumer is whether b is the final consumer.
func (b Buyer) IsFinalConsumer() bool { return b.CustomerID == nil }
