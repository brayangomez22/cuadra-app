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
}

// FinalConsumer is the buyer of a sale without a customer.
func FinalConsumer() Buyer {
	return Buyer{
		PersonType: "natural", DocumentType: FinalConsumerDocumentType,
		DocumentNumber: FinalConsumerDocument, Name: FinalConsumerName,
	}
}

// IsFinalConsumer is whether b is the final consumer.
func (b Buyer) IsFinalConsumer() bool { return b.CustomerID == nil }
