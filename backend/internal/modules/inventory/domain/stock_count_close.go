package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CountAdjustmentReason is the reason of the adjustments an approved count
// records.
const CountAdjustmentReason = "Conteo físico"

// CountAdjustment is how much a variant's stock must change after a count:
// counted − expected.
type CountAdjustment struct {
	VariantID uuid.UUID
	Delta     decimal.Decimal
}

// Approve closes the count with the evaluated results of its lines and
// returns the adjustments to record, one per line with a difference, in
// variant order (the order every stock change locks levels in). Uncounted
// lines are not adjusted. It fails with ErrCountNeedsReview if a line is
// under review, and with ErrCountNotOpen if the count is closed; then the
// count does not change.
func (c *StockCount) Approve(results []LineResult, userID uuid.UUID, at time.Time) ([]CountAdjustment, error) {
	if err := c.close(userID, at); err != nil {
		return nil, err
	}
	var adjustments []CountAdjustment
	for _, r := range results {
		switch r.Status {
		case LineReview:
			return nil, ErrCountNeedsReview
		case LineDifference:
			adjustments = append(adjustments, CountAdjustment{VariantID: r.VariantID, Delta: r.Difference})
		}
	}
	slices.SortFunc(adjustments, func(a, b CountAdjustment) int { return compareIDs(a.VariantID, b.VariantID) })
	summary := Summarize(results)
	c.status, c.closedBy, c.closedAt, c.summary = CountApproved, userID, at.UTC(), &summary
	return adjustments, nil
}

// Cancel closes the count without adjusting anything.
func (c *StockCount) Cancel(userID uuid.UUID, at time.Time) error {
	if err := c.close(userID, at); err != nil {
		return err
	}
	c.status, c.closedBy, c.closedAt = CountCanceled, userID, at.UTC()
	return nil
}

// close checks that the count can be closed by userID.
func (c *StockCount) close(userID uuid.UUID, at time.Time) error {
	if err := c.CheckOpen(); err != nil {
		return err
	}
	if userID == uuid.Nil || at.IsZero() {
		return ErrInvalidUserID
	}
	return nil
}

// AdjustmentInfo is who records the adjustments of the count and when; they
// reference the count, so the kardex leads back to it.
func (c *StockCount) AdjustmentInfo(userID uuid.UUID, at time.Time) MovementInfo {
	return MovementInfo{
		UserID:     userID,
		Reference:  &DocumentRef{Kind: DocumentStockCount, ID: c.id},
		OccurredAt: at,
	}
}
