package app

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// pollLimit is how many tenants, and documents of each, one poll looks up.
const pollLimit = 100

// PollSent looks up, across tenants, the documents the provider received
// while the DIAN was not available, and records the DIAN's answer of those
// it already has. A failure with a document does not stop the others; they
// are all returned. It returns how many documents it updated.
func (s *Service) PollSent(ctx context.Context) (_ int, err error) {
	ctx, span := s.spans.Start(ctx, "PollSent")
	defer func() { usecase.End(span, err) }()

	tenants, err := s.Documents.SentTenants(ctx, pollLimit)
	if err != nil {
		return 0, err
	}
	updated := 0
	var errs []error
	for _, tenantID := range tenants {
		n, err := s.pollTenant(ctx, tenantID)
		updated += n
		errs = append(errs, err)
	}
	span.SetAttributes(attribute.Int("documents.updated", updated))
	return updated, errors.Join(errs...)
}

// pollTenant looks up the tenant's sent documents.
func (s *Service) pollTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	docs, err := s.Documents.Sent(ctx, tenantID, pollLimit)
	if err != nil {
		return 0, err
	}
	updated := 0
	var errs []error
	for _, doc := range docs {
		res, found, err := s.Invoicer.Lookup(ctx, refOf(doc))
		if err != nil {
			s.countProviderError(ctx, classify(err))
			errs = append(errs, err)
			continue
		}
		if !found || res.Status == domain.StatusSent {
			continue
		}
		if err := s.apply(ctx, doc, res); err != nil {
			errs = append(errs, err)
			continue
		}
		updated++
	}
	return updated, errors.Join(errs...)
}
