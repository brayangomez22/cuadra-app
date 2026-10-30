package demo

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/brayangomez22/cuadra-app/backend/internal/demo/apiclient"
)

const (
	// countedLines is how many products each demo count covers, of the
	// location's whole list: a count that was left partial, as many are.
	countedLines = 40
	// shortEvery is how often a counted product comes up short by one.
	shortEvery = 8
)

// writeStockCounts leaves an approved physical count per location after the
// last day's closing, so the demo shows its accuracy report. It goes after the
// history: the plan of the days does not know its adjustments.
func (r *runner) writeStockCounts(ctx context.Context) error {
	r.clock.Set(r.s.Days[len(r.s.Days)-1].Closing.Add(30 * time.Minute))
	for _, location := range r.ids.locations {
		if err := r.stockCount(ctx, apiclient.StockCountInput{LocationId: location}); err != nil {
			return err
		}
	}
	return nil
}

// stockCount starts a count, records the first products as the owner and
// approves it.
func (r *runner) stockCount(ctx context.Context, in apiclient.StockCountInput) error {
	owner := r.as(r.s.Profile.OwnerEmail())
	started, err := r.api.StartStockCountWithResponse(ctx, in, owner)
	if err := expect("start stock count", started, err); err != nil {
		return err
	}
	id := started.JSON201.Id
	limit := countedLines
	lines, err := r.api.ListStockCountLinesWithResponse(ctx, id, &apiclient.ListStockCountLinesParams{Limit: &limit}, owner)
	if err := expect("list stock count lines", lines, err); err != nil {
		return err
	}
	for i, line := range lines.JSON200.Items {
		counted, err := countedOf(line.ExpectedQuantity, i)
		if err != nil {
			return err
		}
		res, err := r.api.RecordCountEntryWithResponse(ctx, id, apiclient.CountEntryInput{VariantId: line.VariantId, Quantity: counted}, owner)
		if err := expect("record count entry", res, err); err != nil {
			return err
		}
	}
	approved, err := r.api.ApproveStockCountWithResponse(ctx, id, owner)
	return expect("approve stock count", approved, err)
}

// countedOf is what was found of the i-th product: what the system expects,
// or one less every so often.
func countedOf(expected string, i int) (string, error) {
	q, err := decimal.NewFromString(expected)
	if err != nil {
		return "", err
	}
	if i%shortEvery == 0 && q.GreaterThanOrEqual(decimal.NewFromInt(1)) {
		q = q.Sub(decimal.NewFromInt(1))
	}
	return q.String(), nil
}
