package app

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/brayangomez22/cuadra-app/backend/internal/modules/inventory/domain"
	"github.com/brayangomez22/cuadra-app/backend/internal/platform/usecase"
)

// LineFilter selects the evaluated lines of a count, optionally of one
// status.
type LineFilter struct {
	Status *domain.LineStatus
	Limit  int
	Offset int
}

// LineView is an evaluated line with the product of its variant.
type LineView struct {
	domain.LineResult
	ProductID uuid.UUID
}

// CountLinesPage is one page of evaluated lines, how many match in total and
// the summary of every line of the count.
type CountLinesPage struct {
	Items   []LineView
	Total   int
	Summary domain.CountSummary
}

// StockCountLines evaluates every line of a count (expected, counted and
// difference) and returns one page of those matching f: the lines under
// review first, then the largest differences in value.
func (s *Service) StockCountLines(ctx context.Context, a Actor, id uuid.UUID, f LineFilter) (_ CountLinesPage, err error) {
	ctx, span := s.startSpan(ctx, "StockCountLines", a, countAttr(id))
	defer func() { usecase.End(span, err) }()

	c, err := s.Counts.Get(ctx, a.TenantID, id)
	if err != nil {
		return CountLinesPage{}, err
	}
	results, err := s.evaluate(ctx, c)
	if err != nil {
		return CountLinesPage{}, err
	}
	span.SetAttributes(attribute.Int("lines", len(results)))
	page := CountLinesPage{Summary: domain.Summarize(results)}
	matching := slices.DeleteFunc(results, func(r domain.LineResult) bool { return f.Status != nil && r.Status != *f.Status })
	slices.SortStableFunc(matching, byRelevance)
	page.Total = len(matching)
	shown := matching[min(f.Offset, len(matching)):min(f.Offset+f.Limit, len(matching))]

	page.Items, err = withProductOf(ctx, s, a, shown, func(r domain.LineResult) uuid.UUID { return r.VariantID },
		func(r domain.LineResult, product uuid.UUID) LineView {
			return LineView{LineResult: r, ProductID: product}
		})
	if err != nil {
		return CountLinesPage{}, err
	}
	return page, nil
}

// byRelevance puts the lines under review first and then the largest
// differences in value; the sheet's variant order breaks ties.
func byRelevance(a, b domain.LineResult) int {
	if review := (a.Status == domain.LineReview); review != (b.Status == domain.LineReview) {
		if review {
			return -1
		}
		return 1
	}
	return b.Value.Abs().Cmp(a.Value.Abs())
}

// evaluate compares what was counted of each line of c with what was
// expected.
func (s *Service) evaluate(ctx context.Context, c *domain.StockCount) ([]domain.LineResult, error) {
	sheet, err := s.Counts.Sheet(ctx, c)
	if err != nil {
		return nil, err
	}
	results := make([]domain.LineResult, 0, len(sheet))
	for _, l := range sheet {
		results = append(results, domain.EvaluateLine(l, c.Mode()))
	}
	return results, nil
}

// RequestRecount starts a new round of the lines of variantIDs in an open
// count: what was counted of them so far no longer counts.
func (s *Service) RequestRecount(ctx context.Context, a Actor, id uuid.UUID, variantIDs []uuid.UUID) (err error) {
	ctx, span := s.startSpan(ctx, "RequestRecount", a, countAttr(id), attribute.Int("lines", len(variantIDs)))
	defer func() { usecase.End(span, err) }()

	c, err := s.Counts.Get(ctx, a.TenantID, id)
	if err != nil {
		return err
	}
	if err := c.CheckOpen(); err != nil {
		return err
	}
	return s.Counts.Recount(ctx, a.TenantID, id, variantIDs)
}
