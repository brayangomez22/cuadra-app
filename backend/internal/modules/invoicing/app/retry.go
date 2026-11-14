package app

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// RetryDocument sends a pending or rejected document again, by hand, with
// the same number, once its data was fixed: it enqueues its issue in the
// same transaction that stores it.
func (s *Service) RetryDocument(ctx context.Context, a Actor, id uuid.UUID) (_ *domain.Document, err error) {
	ctx, span := s.startSpan(ctx, "RetryDocument", a, documentAttr(id))
	defer func() { usecase.End(span, err) }()

	doc, err := s.Documents.Get(ctx, a.TenantID, id)
	if err != nil {
		return nil, err
	}
	before := doc.Status()
	if err := doc.Retry(s.Now()); err != nil {
		return nil, err
	}
	if err := s.Documents.Requeue(ctx, doc); err != nil {
		return nil, err
	}
	s.retries.Add(ctx, 1, metric.WithAttributes(attribute.String("document.status", string(before))))
	return doc, nil
}
