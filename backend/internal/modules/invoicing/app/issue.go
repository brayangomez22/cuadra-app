package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// IssueDocument sends a pending document to the provider, for the
// background job. It runs again and again until it gets an answer: a
// document already answered, or with an error of its data, is left alone;
// one that was sent before is first looked up by its number, so a lost
// answer never issues it twice. A temporary failure of the provider is
// returned for the job to retry later; a request the provider refuses leaves
// the document pending with its error, for a person to retry.
func (s *Service) IssueDocument(ctx context.Context, tenantID, id uuid.UUID) (err error) {
	ctx, span := s.spans.Start(ctx, "IssueDocument", attribute.String("tenant.id", tenantID.String()), documentAttr(id))
	defer func() { usecase.End(span, err) }()

	doc, err := s.Documents.Get(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if doc.Status() != domain.StatusPending || doc.LastError() != "" {
		return nil
	}
	if done, err := s.lookUpFirst(ctx, doc); done || err != nil {
		return err
	}
	return s.send(ctx, doc)
}

// lookUpFirst looks a document sent before up by its number, and records the
// answer the provider already has. done is false when it must be sent.
func (s *Service) lookUpFirst(ctx context.Context, doc *domain.Document) (done bool, _ error) {
	if doc.Attempts() == 0 {
		return false, nil
	}
	res, found, err := s.Invoicer.Lookup(ctx, refOf(doc))
	if err != nil {
		return true, s.providerFailed(ctx, doc, err)
	}
	// A rejection found is the one a person already saw: the document was
	// retried by hand to be sent again, with the same number.
	if !found || res.Status == domain.StatusRejected {
		return false, nil
	}
	return true, s.apply(ctx, doc, res)
}

// send sends the document to the provider and records its answer.
func (s *Service) send(ctx context.Context, doc *domain.Document) error {
	req, err := s.issueRequest(ctx, doc)
	if err != nil {
		return err
	}
	if err := doc.BeginAttempt(s.Now()); err != nil {
		return err
	}
	// The attempt is stored before sending: if the answer is lost, the next
	// attempt looks the document up instead of sending it again.
	if err := s.Documents.Save(ctx, doc); err != nil {
		return err
	}
	res, err := s.Invoicer.Issue(ctx, req)
	if err != nil {
		return s.providerFailed(ctx, doc, err)
	}
	return s.apply(ctx, doc, res)
}

// issueRequest reads the document's sale and buyer.
func (s *Service) issueRequest(ctx context.Context, doc *domain.Document) (IssueRequest, error) {
	sale, err := s.Sales.Sale(ctx, doc.TenantID(), doc.CreatedBy(), doc.SaleID())
	if err != nil {
		return IssueRequest{}, fmt.Errorf("invoicing: read sale: %w", err)
	}
	buyer := domain.FinalConsumer()
	if sale.CustomerID != nil {
		if buyer, err = s.Buyers.Buyer(ctx, doc.TenantID(), doc.CreatedBy(), *sale.CustomerID); err != nil {
			return IssueRequest{}, fmt.Errorf("invoicing: read buyer: %w", err)
		}
	}
	r, err := s.resolutionOf(ctx, doc)
	if err != nil {
		return IssueRequest{}, err
	}
	return IssueRequest{
		DocumentRef: refOf(doc), DocumentID: doc.ID(), Resolution: r.Params(),
		IssuedAt: doc.IssuedAt(), Buyer: buyer, Sale: sale,
	}, nil
}

// resolutionOf returns the resolution that numbered doc.
func (s *Service) resolutionOf(ctx context.Context, doc *domain.Document) (*domain.Resolution, error) {
	rs, err := s.Resolutions.List(ctx, doc.TenantID())
	if err != nil {
		return nil, err
	}
	for _, r := range rs {
		if r.ID() == doc.ResolutionID() {
			return r, nil
		}
	}
	return nil, domain.ErrResolutionNotFound
}

// providerFailed counts a failure of the provider. A temporary one is
// returned for the job to retry; a refused request is stored on the
// document, which waits for a person.
func (s *Service) providerFailed(ctx context.Context, doc *domain.Document, err error) error {
	perr := classify(err)
	s.countProviderError(ctx, perr)
	if perr.Temporary() {
		return err
	}
	s.Logger.ErrorContext(ctx, "provider refused an electronic document",
		slog.String("document_id", doc.ID().String()), slog.String("error", perr.Error()))
	if err := doc.Fail(perr.Error(), s.Now()); err != nil {
		return err
	}
	return s.Documents.Save(ctx, doc)
}

// classify returns the provider's failure of err; an error that is not a
// *ProviderError (the context's) is a timeout.
func classify(err error) *ProviderError {
	var perr *ProviderError
	if !errors.As(err, &perr) {
		perr = &ProviderError{Reason: ReasonTimeout, Err: err}
	}
	return perr
}

// countProviderError counts a failure of the provider by its reason.
func (s *Service) countProviderError(ctx context.Context, perr *ProviderError) {
	s.providerErrors.Add(ctx, 1, metric.WithAttributes(attribute.String("provider.reason", string(perr.Reason))))
}

// apply records the provider's answer on the document.
func (s *Service) apply(ctx context.Context, doc *domain.Document, res Result) error {
	now := s.Now()
	var err error
	switch res.Status {
	case domain.StatusAccepted:
		err = doc.Accept(domain.Acceptance{ProviderID: res.ProviderID, Code: res.Code, QR: res.QR}, now)
	case domain.StatusSent:
		err = doc.MarkSent(res.ProviderID, now)
	case domain.StatusRejected:
		err = doc.Reject(res.Reasons, now)
	default:
		err = fmt.Errorf("invoicing: unknown provider status %q", res.Status)
	}
	if err != nil {
		return err
	}
	if err := s.Documents.Save(ctx, doc); err != nil {
		return err
	}
	s.recordAnswer(ctx, doc)
	return nil
}

// recordAnswer counts the document's new state and, once accepted, how long
// the DIAN took.
func (s *Service) recordAnswer(ctx context.Context, doc *domain.Document) {
	s.countDocument(ctx, doc)
	switch doc.Status() {
	case domain.StatusAccepted:
		s.latency.Record(ctx, doc.AcceptedAt().Sub(doc.IssuedAt()).Seconds(), metric.WithAttributes(kindAttr(doc.Kind())))
	case domain.StatusRejected:
		s.Logger.WarnContext(ctx, "electronic document rejected by the DIAN",
			slog.String("document_id", doc.ID().String()), slog.Int("reasons", len(doc.Reasons())))
	}
}

func refOf(d *domain.Document) DocumentRef {
	return DocumentRef{TenantID: d.TenantID(), Kind: d.Kind(), Prefix: d.Prefix(), Number: d.Number()}
}
