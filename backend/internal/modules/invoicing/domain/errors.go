package domain

import "errors"

// Errors of the resolutions and their numbering.
var (
	ErrInvalidKind             = errors.New("invoicing: invalid document kind")
	ErrInvalidPrefix           = errors.New("invoicing: invalid prefix")
	ErrInvalidResolutionNumber = errors.New("invoicing: invalid resolution number")
	ErrInvalidTechnicalKey     = errors.New("invoicing: invalid technical key")
	ErrInvalidRange            = errors.New("invoicing: invalid numbering range")
	ErrInvalidValidity         = errors.New("invoicing: invalid validity")
	ErrResolutionOverlaps      = errors.New("invoicing: resolution overlaps another")
	ErrResolutionNotFound      = errors.New("invoicing: resolution not found")
	ErrResolutionInUse         = errors.New("invoicing: resolution already numbered documents")
	ErrResolutionNotValid      = errors.New("invoicing: resolution not valid on that day")
	ErrResolutionExhausted     = errors.New("invoicing: resolution range exhausted")
	ErrNoActiveResolution      = errors.New("invoicing: no active resolution")
)

// Errors of the documents.
var (
	ErrInvoiceRequiresCustomer = errors.New("invoicing: an electronic invoice requires a customer")
	ErrDocumentNotFound        = errors.New("invoicing: document not found")
	ErrDocumentNotIssuable     = errors.New("invoicing: document is not pending")
	ErrDocumentNotRetryable    = errors.New("invoicing: document cannot be retried")
	ErrInvalidTransition       = errors.New("invoicing: invalid document state change")
	ErrMissingReasons          = errors.New("invoicing: a rejection needs its reasons")
	ErrMissingCode             = errors.New("invoicing: an accepted document needs its CUDE or CUFE")
)
