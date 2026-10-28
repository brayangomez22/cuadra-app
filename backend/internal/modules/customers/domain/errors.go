package domain

import (
	"errors"
	"fmt"

	"github.com/brayangomez22/cuadra-app/backend/internal/kernel/taxid"
)

// Domain errors of the customers module. The HTTP adapter maps them to
// status codes.
var (
	ErrInvalidTenantID     = errors.New("customers: invalid tenant id")
	ErrInvalidCustomerID   = errors.New("customers: invalid customer id")
	ErrInvalidPersonType   = errors.New("customers: invalid person type")
	ErrInvalidDocumentType = errors.New("customers: invalid document type")
	ErrInvalidDocument     = errors.New("customers: invalid document number for its type")
	// ErrInvalidNITCheckDigit is the shared kernel's: the NIT's DV does not
	// match.
	ErrInvalidNITCheckDigit       = taxid.ErrInvalidNITCheckDigit
	ErrLegalPersonRequiresNIT     = errors.New("customers: a legal person is identified by NIT")
	ErrInvalidCustomerName        = errors.New("customers: invalid customer name")
	ErrInvalidContact             = errors.New("customers: invalid contact data")
	ErrInvalidTaxResponsibilities = errors.New("customers: invalid tax responsibilities")
	ErrInvalidSegmentID           = errors.New("customers: invalid segment id")
	ErrInvalidSegmentName         = errors.New("customers: invalid segment name")

	// Errors reported by the repositories and the use cases. ErrDocumentTaken
	// comes wrapped in a *DocumentTakenError that names the document.
	ErrCustomerNotFound = errors.New("customers: customer not found")
	ErrCustomerInactive = errors.New("customers: customer is inactive")
	ErrDocumentTaken    = errors.New("customers: document already used by another customer of the tenant")
	ErrSegmentNotFound  = errors.New("customers: segment not found")
	ErrSegmentNameTaken = errors.New("customers: segment name already used in the tenant")
	ErrSegmentInUse     = errors.New("customers: segment has customers")
)

// DocumentTakenError is an ErrDocumentTaken that names the document already
// in use.
type DocumentTakenError struct {
	Document Document
}

func (e *DocumentTakenError) Error() string {
	return fmt.Sprintf("%v: %s", ErrDocumentTaken, e.Document)
}

func (e *DocumentTakenError) Unwrap() error { return ErrDocumentTaken }
