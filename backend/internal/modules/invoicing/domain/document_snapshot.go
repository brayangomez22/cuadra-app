package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// DocumentSnapshot is a stored document.
type DocumentSnapshot struct {
	ID, TenantID, SaleID, CreatedBy uuid.UUID
	Kind                            Kind
	ResolutionID                    uuid.UUID
	Prefix                          string
	Number                          int64
	IssuedAt                        time.Time
	Status                          Status
	Attempts                        int
	ProviderID, Code, QR            string
	Reasons                         []string
	LastError                       string
	AcceptedAt                      *time.Time
	UpdatedAt                       time.Time
}

// RestoreDocument rebuilds a stored document.
func RestoreDocument(s DocumentSnapshot) *Document {
	return &Document{
		id: s.ID, tenantID: s.TenantID, saleID: s.SaleID, createdBy: s.CreatedBy,
		kind: s.Kind, resolutionID: s.ResolutionID, prefix: s.Prefix, number: s.Number,
		issuedAt: s.IssuedAt, status: s.Status, attempts: s.Attempts,
		providerID: s.ProviderID, code: s.Code, qr: s.QR, reasons: slices.Clone(s.Reasons),
		lastError: s.LastError, acceptedAt: copyTime(s.AcceptedAt), updatedAt: s.UpdatedAt,
	}
}

// Snapshot returns the document's stored form.
func (d *Document) Snapshot() DocumentSnapshot {
	return DocumentSnapshot{
		ID: d.id, TenantID: d.tenantID, SaleID: d.saleID, CreatedBy: d.createdBy,
		Kind: d.kind, ResolutionID: d.resolutionID, Prefix: d.prefix, Number: d.number,
		IssuedAt: d.issuedAt, Status: d.status, Attempts: d.attempts,
		ProviderID: d.providerID, Code: d.code, QR: d.qr, Reasons: slices.Clone(d.reasons),
		LastError: d.lastError, AcceptedAt: copyTime(d.acceptedAt), UpdatedAt: d.updatedAt,
	}
}

// ID returns the document's id.
func (d *Document) ID() uuid.UUID { return d.id }

// TenantID returns the tenant the document belongs to.
func (d *Document) TenantID() uuid.UUID { return d.tenantID }

// SaleID returns the sale the document is of.
func (d *Document) SaleID() uuid.UUID { return d.saleID }

// CreatedBy returns who confirmed the document's sale.
func (d *Document) CreatedBy() uuid.UUID { return d.createdBy }

// Kind returns the document's kind.
func (d *Document) Kind() Kind { return d.kind }

// ResolutionID returns the resolution that numbered the document.
func (d *Document) ResolutionID() uuid.UUID { return d.resolutionID }

// Prefix returns the document's prefix.
func (d *Document) Prefix() string { return d.prefix }

// Number returns the document's consecutive.
func (d *Document) Number() int64 { return d.number }

// IssuedAt returns when the document was issued: when its sale got it.
func (d *Document) IssuedAt() time.Time { return d.issuedAt }

// Status returns the document's state.
func (d *Document) Status() Status { return d.status }

// Attempts returns how many times it was sent to the provider.
func (d *Document) Attempts() int { return d.attempts }

// ProviderID returns the provider's id of the document, once it has one.
func (d *Document) ProviderID() string { return d.providerID }

// Code returns the CUDE or CUFE of an accepted document.
func (d *Document) Code() string { return d.code }

// QR returns the QR's content of an accepted document.
func (d *Document) QR() string { return d.qr }

// Reasons returns why the DIAN rejected the document.
func (d *Document) Reasons() []string { return slices.Clone(d.reasons) }

// LastError returns the error of the document's data, if the provider
// refused it.
func (d *Document) LastError() string { return d.lastError }

// AcceptedAt returns when the DIAN validated the document.
func (d *Document) AcceptedAt() *time.Time { return copyTime(d.acceptedAt) }

// UpdatedAt returns when the document last changed.
func (d *Document) UpdatedAt() time.Time { return d.updatedAt }

func copyTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	c := *t
	return &c
}
