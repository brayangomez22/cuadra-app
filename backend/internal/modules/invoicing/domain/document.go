package domain

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Status is where a document is on its way to the DIAN.
type Status string

// The states of a document.
const (
	// StatusPending has its number but no answer from the provider yet.
	StatusPending Status = "pending"
	// StatusSent was received by the provider, which waits for the DIAN.
	StatusSent Status = "sent"
	// StatusAccepted was validated by the DIAN: it has its CUDE or CUFE.
	StatusAccepted Status = "accepted"
	// StatusRejected was rejected by the DIAN, with its reasons. It is not
	// retried on its own: the data is fixed and it is sent again by hand.
	StatusRejected Status = "rejected"
)

// DocumentParams are the data of a sale's new document.
type DocumentParams struct {
	ID, TenantID, SaleID uuid.UUID
	// CreatedBy is who confirmed the sale; the background job issues the
	// document on their behalf.
	CreatedBy  uuid.UUID
	Kind       Kind
	Resolution *Resolution
	// Number is the consecutive Resolution.Draw gave.
	Number   int64
	IssuedAt time.Time
}

// Acceptance is what the provider returns for a validated document.
type Acceptance struct {
	ProviderID string
	// Code is the CUDE of a POS document or the CUFE of an invoice.
	Code string
	QR   string
}

// Document is the electronic document of a sale. Its kind, resolution,
// prefix and number never change: retrying it sends the same number, so it
// is never issued twice.
type Document struct {
	id, tenantID, saleID, createdBy uuid.UUID
	kind                            Kind
	resolutionID                    uuid.UUID
	prefix                          string
	number                          int64
	issuedAt                        time.Time
	status                          Status
	attempts                        int
	providerID, code, qr            string
	reasons                         []string
	lastError                       string
	acceptedAt                      *time.Time
	updatedAt                       time.Time
}

// NewDocument returns the pending document of a sale with its number.
func NewDocument(p DocumentParams) (*Document, error) {
	rp := p.Resolution.Params()
	switch {
	case p.Kind != rp.Kind:
		return nil, ErrInvalidKind
	case p.Number < rp.From || p.Number > rp.To:
		return nil, ErrInvalidRange
	}
	return &Document{
		id: p.ID, tenantID: p.TenantID, saleID: p.SaleID, createdBy: p.CreatedBy,
		kind: p.Kind, resolutionID: p.Resolution.ID(), prefix: rp.Prefix, number: p.Number,
		issuedAt: p.IssuedAt, status: StatusPending, updatedAt: p.IssuedAt,
	}, nil
}

// BeginAttempt counts an attempt to send the document, before sending it:
// a later attempt then knows to look it up first. Only a pending document
// without an error can be sent (ErrDocumentNotIssuable).
func (d *Document) BeginAttempt(at time.Time) error {
	if d.status != StatusPending || d.lastError != "" {
		return ErrDocumentNotIssuable
	}
	d.attempts++
	d.updatedAt = at
	return nil
}

// MarkSent records that the provider received the pending document and
// waits for the DIAN.
func (d *Document) MarkSent(providerID string, at time.Time) error {
	if d.status != StatusPending {
		return ErrInvalidTransition
	}
	d.status, d.providerID, d.updatedAt = StatusSent, providerID, at
	return nil
}

// Accept records the DIAN's validation of a pending or sent document.
func (d *Document) Accept(a Acceptance, at time.Time) error {
	if !d.waiting() {
		return ErrInvalidTransition
	}
	if strings.TrimSpace(a.Code) == "" {
		return ErrMissingCode
	}
	if a.ProviderID != "" {
		d.providerID = a.ProviderID
	}
	d.status, d.code, d.qr, d.updatedAt = StatusAccepted, a.Code, a.QR, at
	d.acceptedAt = &at
	return nil
}

// Reject records the DIAN's rejection of a pending or sent document, with
// its reasons.
func (d *Document) Reject(reasons []string, at time.Time) error {
	if !d.waiting() {
		return ErrInvalidTransition
	}
	if len(reasons) == 0 {
		return ErrMissingReasons
	}
	d.status, d.reasons, d.updatedAt = StatusRejected, slices.Clone(reasons), at
	return nil
}

// Fail records an error of the document's data that retrying does not fix
// (the provider refused the request). It stays pending, and is not sent
// again until it is retried by hand.
func (d *Document) Fail(message string, at time.Time) error {
	if d.status != StatusPending {
		return ErrInvalidTransition
	}
	d.lastError, d.updatedAt = message, at
	return nil
}

// Retry readies a pending or rejected document to be sent again, by hand,
// with the same number: it clears its reasons and its error.
func (d *Document) Retry(at time.Time) error {
	if d.status != StatusPending && d.status != StatusRejected {
		return ErrDocumentNotRetryable
	}
	d.status, d.reasons, d.lastError, d.updatedAt = StatusPending, nil, "", at
	return nil
}

// waiting is whether the document waits for the DIAN's answer.
func (d *Document) waiting() bool { return d.status == StatusPending || d.status == StatusSent }
