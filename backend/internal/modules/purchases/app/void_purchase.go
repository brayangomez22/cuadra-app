package app

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/purchases/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// VoidPurchase voids a purchase registered by mistake and takes out the
// stock it put in, in one transaction: if a variant no longer has that stock
// (it was sold), the purchase stays and the error names its first line. A
// purchase on credit with payments is not voided. reason is required.
func (s *Service) VoidPurchase(ctx context.Context, a Actor, id uuid.UUID, reason string) (_ *domain.Purchase, err error) {
	ctx, span := s.startSpan(ctx, "VoidPurchase", a, purchaseAttr(id))
	defer func() { usecase.End(span, err) }()

	p, err := s.Purchases.Void(ctx, a.TenantID, id, func(ctx context.Context, p *domain.Purchase) error {
		if err := p.Void(a.UserID, reason, s.Now()); err != nil {
			return err
		}
		return lineOf(p, s.Stock.ReversePurchase(ctx, a.TenantID, a.UserID, p.ID()))
	})
	if err != nil {
		return nil, err
	}
	s.voided.Add(ctx, 1, metric.WithAttributes(attribute.String("payment", string(p.Payment()))))
	return p, nil
}

// lineOf turns the error of one variant into the error of its first line of
// p, so the client marks it; other errors pass through.
func lineOf(p *domain.Purchase, err error) error {
	var variantErr *VariantError
	if !errors.As(err, &variantErr) {
		return err
	}
	i := slices.IndexFunc(p.Lines(), func(l domain.Line) bool { return l.VariantID() == variantErr.VariantID })
	if i < 0 {
		return variantErr.Err
	}
	return &LineError{Index: i, Err: variantErr.Err}
}
