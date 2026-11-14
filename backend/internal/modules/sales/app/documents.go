package app

import (
	"context"

	"github.com/google/uuid"
)

// ElectronicDocuments gives the sales their electronic document, in the
// invoicing module.
type ElectronicDocuments interface {
	// RegisterSale numbers the document of a sale being confirmed, joining
	// the sale's transaction with its ctx; nothing when the tenant does not
	// issue documents. It fails with domain.ErrNoActiveResolution,
	// domain.ErrInvoiceRequiresCustomer or domain.ErrInvalidDocumentKind.
	RegisterSale(ctx context.Context, req DocumentRequest) error
	// HasDocument is whether the sale has an electronic document.
	HasDocument(ctx context.Context, tenantID, userID, saleID uuid.UUID) (bool, error)
}

// DocumentRequest is a sale that may need an electronic document.
type DocumentRequest struct {
	TenantID, UserID, SaleID uuid.UUID
	// Kind is the kind the sale asks for ("pos" or "invoice"), or "".
	Kind        string
	Credit      bool
	HasCustomer bool
}
