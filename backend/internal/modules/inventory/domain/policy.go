package domain

// StockPolicy holds the tenant's stock rules. The zero value is the default:
// stock cannot go negative.
type StockPolicy struct {
	// AllowNegative lets sales, negative adjustments and transfers take more
	// than the stock on hand, for stores that sell before receiving.
	AllowNegative bool
}
