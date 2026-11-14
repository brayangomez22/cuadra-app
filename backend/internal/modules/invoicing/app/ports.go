package app

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// ElectronicInvoicer is the technology provider that signs the documents,
// sends them to the DIAN and returns its answer. Its adapter translates the
// documents to the provider's API.
type ElectronicInvoicer interface {
	// Issue sends a document. A DIAN rejection is a Result, not an error;
	// an error is a *ProviderError (or one of the context).
	Issue(ctx context.Context, req IssueRequest) (Result, error)
	// Lookup finds a document the provider already received, by its kind,
	// prefix and number: found is false if it never got it.
	Lookup(ctx context.Context, ref DocumentRef) (_ Result, found bool, _ error)
}

// DocumentRef names a document at the provider: its number is unique.
type DocumentRef struct {
	TenantID uuid.UUID
	Kind     domain.Kind
	Prefix   string
	Number   int64
}

// IssueRequest is a document to issue, with the data of its sale.
type IssueRequest struct {
	DocumentRef
	DocumentID uuid.UUID
	Resolution domain.ResolutionParams
	IssuedAt   time.Time
	Buyer      domain.Buyer
	Sale       SaleData
}

// Result is the provider's answer about a document: accepted (with its
// code and QR), sent (waiting for the DIAN) or rejected (with its reasons).
type Result struct {
	Status     domain.Status
	ProviderID string
	Code       string
	QR         string
	Reasons    []string
}

// ProviderErrorReason classifies a failure of the provider.
type ProviderErrorReason string

// The reasons of a provider's failure.
const (
	ReasonTimeout     ProviderErrorReason = "timeout"
	ReasonRateLimited ProviderErrorReason = "rate_limited"
	ReasonServer      ProviderErrorReason = "server"
	// ReasonBadRequest is a request the provider refused: a bug of Cuadra
	// or bad data, which retrying does not fix.
	ReasonBadRequest ProviderErrorReason = "bad_request"
)

// ProviderError is a failure to talk to the provider.
type ProviderError struct {
	Reason ProviderErrorReason
	Err    error
}

func (e *ProviderError) Error() string {
	return "invoicing: provider " + string(e.Reason) + ": " + e.Err.Error()
}

func (e *ProviderError) Unwrap() error { return e.Err }

// Temporary is whether trying again later may work.
func (e *ProviderError) Temporary() bool { return e.Reason != ReasonBadRequest }

// SaleSource reads a confirmed sale of the sales module.
type SaleSource interface {
	Sale(ctx context.Context, tenantID, userID, saleID uuid.UUID) (SaleData, error)
}

// SaleData is what the document says of its sale. Amounts are in COP,
// already rounded by the sale.
type SaleData struct {
	Number     int64
	LocationID uuid.UUID
	CustomerID *uuid.UUID
	Lines      []SaleLine
	Payments   []SalePayment
	Subtotal   decimal.Decimal
	Discounts  decimal.Decimal
	Tax        decimal.Decimal
	Total      decimal.Decimal
}

// SaleLine is a line of the sale: price before IVA and discount in COP.
type SaleLine struct {
	VariantID uuid.UUID
	Name      string
	Quantity  decimal.Decimal
	UnitPrice decimal.Decimal
	Discount  decimal.Decimal
	TaxRate   decimal.Decimal
	Tax       decimal.Decimal
	Total     decimal.Decimal
}

// SalePayment is a payment of the sale, by its method's code.
type SalePayment struct {
	Method string
	Amount decimal.Decimal
}

// BuyerDirectory reads who a customer is, from the customers module.
type BuyerDirectory interface {
	Buyer(ctx context.Context, tenantID, userID, customerID uuid.UUID) (domain.Buyer, error)
}
