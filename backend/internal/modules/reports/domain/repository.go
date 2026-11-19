package domain

import "context"

// SalesReader reads the sales reports. Every method reads only the
// filter's tenant (inside its Row-Level Security transaction) and counts
// only confirmed sales.
type SalesReader interface {
	// Sales fills the rows and the totals of the sales report.
	Sales(ctx context.Context, f SalesFilter) (SalesReport, error)
	// SoldVariants returns what each variant sold in the period.
	SoldVariants(ctx context.Context, f ProductFilter) ([]SoldVariant, error)
	// SaleProfits fills a page of the sales with their profit, newest first.
	SaleProfits(ctx context.Context, f SaleProfitFilter) (SaleProfitsReport, error)
}

// StockReader reads the stock reports.
type StockReader interface {
	// IdleVariants returns the variants with stock not sold since f.Since.
	IdleVariants(ctx context.Context, f IdleFilter) ([]IdleLine, error)
	// Adjustments fills the rows of the stock adjustments report.
	Adjustments(ctx context.Context, f AdjustmentFilter) (AdjustmentsReport, error)
}

// TeamReader reads the reports of what the team did.
type TeamReader interface {
	// Closings returns the cash sessions closed in the period.
	Closings(ctx context.Context, f CashFilter) ([]Closing, error)
	// TeamControl returns each user's figures in each bucket with any.
	TeamControl(ctx context.Context, f TeamFilter) ([]TeamPoint, error)
}
