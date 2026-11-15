package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/invoicing/domain"
)

// invoicerFor returns the provider of the tenant's documents and the
// company that issues them. A demo tenant's documents go to the simulated
// provider, whatever the process's provider is: they must never be real.
func (s *Service) invoicerFor(ctx context.Context, tenantID uuid.UUID) (ElectronicInvoicer, domain.Issuer, error) {
	issuer, err := s.Issuers.Issuer(ctx, tenantID)
	if err != nil {
		return nil, domain.Issuer{}, fmt.Errorf("invoicing: read issuer: %w", err)
	}
	if issuer.Demo {
		return s.DemoInvoicer, issuer, nil
	}
	return s.Invoicer, issuer, nil
}

// storeFiles keeps the files of an accepted document.
func (s *Service) storeFiles(ctx context.Context, doc *domain.Document, files []domain.File) error {
	for _, f := range files {
		if err := s.Files.Put(ctx, doc.TenantID(), doc.ID(), f); err != nil {
			return fmt.Errorf("invoicing: store %s file: %w", f.Kind, err)
		}
		s.files.Add(ctx, 1, metric.WithAttributes(attribute.String("file.kind", string(f.Kind))))
	}
	return nil
}
