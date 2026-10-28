package app

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// StartCountInput is a new stock count at a location: of the variants of
// CategoryIDs (and their subcategories), or of every variant with stock there
// when it is empty.
type StartCountInput struct {
	LocationID  uuid.UUID
	CategoryIDs []uuid.UUID
	Mode        domain.CountMode
}

// CountView is a stock count with how far it went. It never carries the
// expected stock: counters see it, and the count is blind.
type CountView struct {
	*domain.StockCount
	Progress domain.CountProgress
}

// StartStockCount opens a count at an active location and takes the
// snapshot of its stock. The location must have no other open count.
func (s *Service) StartStockCount(ctx context.Context, a Actor, in StartCountInput) (_ CountView, err error) {
	ctx, span := s.startSpan(ctx, "StartStockCount", a, locationAttr(in.LocationID))
	defer func() { usecase.End(span, err) }()

	if err := s.requireActiveLocation(ctx, a, in.LocationID); err != nil {
		return CountView{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return CountView{}, err
	}
	c, err := domain.NewStockCount(domain.StockCountSpec{
		ID: id, TenantID: a.TenantID, LocationID: in.LocationID, CategoryIDs: in.CategoryIDs,
		Mode: in.Mode, StartedBy: a.UserID, StartedAt: s.Now(),
	})
	if err != nil {
		return CountView{}, err
	}
	var variantIDs []uuid.UUID
	if c.IsPartial() {
		if variantIDs, err = s.Variants.VariantsInCategories(ctx, a.TenantID, c.CategoryIDs()); err != nil {
			return CountView{}, err
		}
	}
	lines, err := s.Counts.Start(ctx, c, variantIDs)
	if err != nil {
		return CountView{}, err
	}
	span.SetAttributes(countAttr(c.ID()), attribute.Int("lines", lines))
	s.countCounts(ctx, "started")
	return CountView{StockCount: c, Progress: domain.CountProgress{Lines: lines}}, nil
}

// GetStockCount returns a count of the actor's tenant with its progress.
func (s *Service) GetStockCount(ctx context.Context, a Actor, id uuid.UUID) (_ CountView, err error) {
	ctx, span := s.startSpan(ctx, "GetStockCount", a, countAttr(id))
	defer func() { usecase.End(span, err) }()

	c, err := s.Counts.Get(ctx, a.TenantID, id)
	if err != nil {
		return CountView{}, err
	}
	progress, err := s.Counts.Progress(ctx, a.TenantID, id)
	if err != nil {
		return CountView{}, err
	}
	return CountView{StockCount: c, Progress: progress}, nil
}

// ListStockCounts returns one page of the tenant's counts, newest first;
// the approved ones carry their accuracy.
func (s *Service) ListStockCounts(ctx context.Context, a Actor, f domain.CountFilter) (_ domain.StockCountPage, err error) {
	ctx, span := s.startSpan(ctx, "ListStockCounts", a)
	defer func() { usecase.End(span, err) }()

	return s.Counts.List(ctx, a.TenantID, f)
}

// CancelStockCount closes an open count without adjusting the stock.
func (s *Service) CancelStockCount(ctx context.Context, a Actor, id uuid.UUID) (_ *domain.StockCount, err error) {
	ctx, span := s.startSpan(ctx, "CancelStockCount", a, countAttr(id))
	defer func() { usecase.End(span, err) }()

	var canceled *domain.StockCount
	err = s.Counts.Close(ctx, a.TenantID, id, func(_ context.Context, c *domain.StockCount) error {
		canceled = c
		return c.Cancel(a.UserID, s.Now())
	})
	if err != nil {
		return nil, err
	}
	s.countCounts(ctx, "canceled")
	return canceled, nil
}

func (s *Service) countCounts(ctx context.Context, result string) {
	s.stockCounts.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

func countAttr(id uuid.UUID) attribute.KeyValue {
	return attribute.String("stock_count.id", id.String())
}
