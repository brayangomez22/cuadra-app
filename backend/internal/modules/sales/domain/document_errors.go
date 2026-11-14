package domain

import "errors"

// Errors of a sale's electronic document, which the invoicing module issues.
var (
	// ErrNoActiveResolution means the tenant issues electronic documents
	// but has no numbering resolution in force for the sale's kind: the
	// sale is not confirmed, since it could not be numbered legally.
	ErrNoActiveResolution = errors.New("sales: no numbering resolution in force")
	// ErrInvoiceRequiresCustomer means the sale asked for an electronic
	// invoice without a customer.
	ErrInvoiceRequiresCustomer = errors.New("sales: an electronic invoice requires a customer")
	// ErrInvalidDocumentKind means the sale asked for an unknown kind of
	// electronic document.
	ErrInvalidDocumentKind = errors.New("sales: invalid electronic document kind")
	// ErrSaleInvoiced means the sale has an electronic document: it is not
	// voided, it gets a credit note.
	ErrSaleInvoiced = errors.New("sales: the sale has an electronic document")
)
