package app

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// ApproveStockCount closes an open count and adjusts the stock of every
// line with a difference (adjustment, reason «Conteo físico», referencing
// the count), all in one transaction: if one adjustment fails, nothing is
// adjusted and the count stays open. The adjustments follow the tenant's
// stock policy; one that would leave the stock negative fails with a
// VariantStockError naming the variant, to recount it.
func (s *Service) ApproveStockCount(ctx context.Context, a Actor, id uuid.UUID) (_ *domain.StockCount, err error) {
	ctx, span := s.startSpan(ctx, "ApproveStockCount", a, countAttr(id))
	defer func() { usecase.End(span, err) }()

	policy, err := s.Policies.Get(ctx, a.TenantID)
	if err != nil {
		return nil, err
	}
	var (
		approved  *domain.StockCount
		movements []*domain.StockMovement
	)
	err = s.Counts.Close(ctx, a.TenantID, id, func(ctx context.Context, c *domain.StockCount) error {
		approved = c
		results, err := s.evaluate(ctx, c)
		if err != nil {
			return err
		}
		now := s.Now()
		adjustments, err := c.Approve(results, a.UserID, now)
		if err != nil {
			return err
		}
		movements, err = s.adjustCounted(ctx, c, adjustments, policy, c.AdjustmentInfo(a.UserID, now))
		return err
	})
	if err != nil {
		return nil, err
	}
	span.SetAttributes(attribute.Int("adjustments", len(movements)))
	s.countMovements(ctx, movements...)
	s.countCounts(ctx, "approved")
	if summary, _ := approved.Summary(); summary.Counted > 0 {
		accuracy, _ := summary.Accuracy()
		s.countAccuracy.Record(ctx, accuracy.InexactFloat64())
	}
	return approved, nil
}

// adjustCounted records the adjustments of an approved count, in the order
// given (by variant), within the count's transaction.
func (s *Service) adjustCounted(ctx context.Context, c *domain.StockCount, adjustments []domain.CountAdjustment,
	policy domain.StockPolicy, info domain.MovementInfo,
) ([]*domain.StockMovement, error) {
	movements := make([]*domain.StockMovement, 0, len(adjustments))
	for _, adj := range adjustments {
		err := s.Stock.Apply(ctx, c.TenantID(), adj.VariantID, []uuid.UUID{c.LocationID()},
			func(levels []*domain.StockLevel) ([]*domain.StockMovement, error) {
				m, err := levels[0].Adjust(adj.Delta, domain.CountAdjustmentReason, policy, info)
				if err != nil {
					return nil, err
				}
				movements = append(movements, m)
				return []*domain.StockMovement{m}, nil
			})
		if err != nil {
			s.countInsufficient(ctx, err, domain.MovementAdjustment)
			return nil, &VariantStockError{VariantID: adj.VariantID, Err: err}
		}
	}
	return movements, nil
}
